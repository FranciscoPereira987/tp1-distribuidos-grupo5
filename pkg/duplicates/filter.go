package duplicates

import (
	"bytes"
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	log "github.com/sirupsen/logrus"
)

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
	m, err := stateMan.GetMapStringInt64("last-received")
	if err != nil && errors.Is(err, state.ErrNotFound) {
		log.Infof("No state to recover duplicate filter from")
		return nil
	}
	log.Infof("recovered duplicate filter: %v", m)
	df.lastMessages = m
	return err
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
