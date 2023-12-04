package duplicates

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
)

var ErrUnsupported = errors.New("unsupported value")

type DuplicateFilter struct {
	lastMessages map[string]int64
}

func NewDuplicateFilter() *DuplicateFilter {

	return &DuplicateFilter{
		lastMessages: make(map[string]int64),
	}
}

func (df DuplicateFilter) RemoveFromState(stateMan *state.StateManager) {
	delete(stateMan.State, "last-received")
}

func (df DuplicateFilter) AddToState(stateMan *state.StateManager) {
	stateMan.State["last-received"] = df.lastMessages
}

func (df *DuplicateFilter) RecoverFromState(stateMan *state.StateManager) error {
	value, ok := stateMan.State["last-received"]
	if !ok {
		log.Infof("No state to recover duplicate messages from")
		return nil
	}
	state, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("%w: %#v is not a map", ErrUnsupported, state)
	}

	for key, val := range state {
		num, ok := val.(json.Number)
		if !ok {
			return fmt.Errorf("%w: %#v is not a json.Number", ErrUnsupported, val)
		}
		if lastId, err := num.Int64(); err != nil {
			return fmt.Errorf("recovering duplicate filter: %w", err)
		} else {
			df.lastMessages[key] = lastId
		}
	}
	log.Infof("Recovered from state: %v", df.lastMessages)
	return nil
}

func (df *DuplicateFilter) Update(r *bytes.Reader) (dup bool, err error) {
	var h typing.BatchHeader
	if err = h.Unmarshal(r); err == nil {
		if lastId, ok := df.lastMessages[h.WorkerId]; ok {
			dup = lastId == h.MessageId
		}
		df.lastMessages[h.WorkerId] = h.MessageId
	}
	return dup, err
}
