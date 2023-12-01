package duplicates

import (
	"bytes"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

type DuplicateFilter struct {
	lastMessages map[string]string
	LastMessage string
}

func NewDuplicateFilter() *DuplicateFilter {

	return &DuplicateFilter{
		lastMessages: make(map[string]string),
	}
}

func (df DuplicateFilter) RemoveFromState(stateMan *state.StateManager) {
	delete(stateMan.State, "last-received")
}

func (df DuplicateFilter) AddToState(stateMan *state.StateManager) {
	stateMan.State["last-received"] = df.lastMessages
}

func (df *DuplicateFilter) RecoverFromState(stateMan *state.StateManager) {
	value, ok := stateMan.State["last-received"].(map[string]string)
	if ok {
		df.lastMessages = value
	}
}

func (df *DuplicateFilter) Update(r *bytes.Reader) (dup bool, err error) {
	var h typing.BatchHeader
	if err = h.Unmarshal(r); err == nil {
		dup = df.lastMessages[h.WorkerId] == h.MessageId
		df.lastMessages[h.WorkerId] = h.MessageId
		df.LastMessage = h.MessageId
	}
	return dup, err
}
