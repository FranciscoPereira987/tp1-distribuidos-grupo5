package duplicates

import (
	"bytes"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

const (
	MaxMessages = 1000
)

type DuplicateFilter struct {
	lastMessages map[string]bool
	queueLast    []string

	LastMessage string
}

func NewDuplicateFilter(lastMessage string) *DuplicateFilter {

	return &DuplicateFilter{
		lastMessages: make(map[string]bool),
		queueLast:    make([]string, 0),
		LastMessage:  string(lastMessage),
	}
}

func (df DuplicateFilter) AddToState(stateMan *state.StateManager) {
	stateMan.AddToState("last-received", df.LastMessage)
}

func (df *DuplicateFilter) RecoverFromState(stateMan *state.StateManager) {
	if value := stateMan.GetString("last-received"); value != "" {
		df.LastMessage = value
	}
}

func (df *DuplicateFilter) manageSizes() {
	if len(df.queueLast) >= MaxMessages {
		delete(df.lastMessages, df.queueLast[0])
		df.queueLast = df.queueLast[1:]
	}
}

func (df *DuplicateFilter) ChangeLast(newLastMessage []byte) (*bytes.Reader, error) {
	r := bytes.NewReader(newLastMessage)
	h, err := typing.HeaderUnmarshall(r)

	if err == nil {
		df.lastMessages[h.ID] = true
		df.queueLast = append(df.queueLast, h.ID)
		df.LastMessage = h.ID
		df.manageSizes()
	}

	return r, err
}

func (df DuplicateFilter) IsDuplicate(body []byte) (dup bool) {
	r := bytes.NewReader(body)
	h, err := typing.HeaderUnmarshall(r)
	if err == nil {
		dup = h.ID == df.LastMessage
	}
	return
}
