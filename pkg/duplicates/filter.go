package duplicates

import (
	"bytes"
	"context"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

type DuplicateFilterConfig struct {
	Ctx        context.Context
	Mid        *middleware.Middleware
	StreamName string
	StateFile  string
}

type DuplicateFilter struct {
	LastMessage string
}

func NewDuplicateFilter(lastMessage string) *DuplicateFilter {
	return &DuplicateFilter{
		LastMessage: string(lastMessage),
	}
}

func (df DuplicateFilter) AddToState(stateMan *state.StateManager) {
	stateMan.AddToState("last-received", df.LastMessage)
}

func (df *DuplicateFilter) RecoverFromState(stateMan *state.StateManager) {
	if value := stateMan.Get("last-received"); value != nil {
		df.LastMessage = value.(string)
	}
}

func (df *DuplicateFilter) ChangeLast(newLastMessage []byte) (*bytes.Reader, error) {
	r := bytes.NewReader(newLastMessage)
	h, err := typing.HeaderUnmarshal(r)
	df.LastMessage = h.ID

	return r, err
}

func (df DuplicateFilter) IsDuplicate(body []byte) (dup bool) {
	r := bytes.NewReader(body)
	h, err := typing.HeaderUnmarshal(r)
	if err == nil {
		dup = h.ID == df.LastMessage
	}
	return
}
