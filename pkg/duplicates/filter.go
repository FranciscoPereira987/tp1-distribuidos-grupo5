package duplicates

import (
	"bytes"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

type DuplicateFilter struct {
	lastMessages map[string]uint64
}

func NewDuplicateFilter() *DuplicateFilter {

	return &DuplicateFilter{
		lastMessages: make(map[string]uint64),
	}
}

func (df DuplicateFilter) RemoveFromState(stateMan *state.StateManager) {
	delete(stateMan.State, "last-received")
}

func (df DuplicateFilter) AddToState(stateMan *state.StateManager) {
	stateMan.State["last-received"] = df.lastMessages
}

func (df *DuplicateFilter) RecoverFromState(stateMan *state.StateManager) {
	value, ok := stateMan.State["last-received"].(map[string]uint64)
	if ok {
		df.lastMessages = value
	}
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
