package duplicates

import (
	"bytes"
	"context"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
)

type DuplicateFilterConfig struct {
	Ctx        context.Context
	Mid        *middleware.Middleware
	StreamName string
	StateFile  string
}

type DuplicateFilter struct {
	lastMessage []byte
}

func NewDuplicateFilter(lastMessage []byte) *DuplicateFilter {
	return &DuplicateFilter{
		lastMessage: lastMessage,
	}
}

func (df DuplicateFilter) AddToState(stateMan *state.StateManager) {
	stateMan.AddToState("last-received", df.lastMessage)
}

func (df *DuplicateFilter) ChangeLast(newLastMessage []byte) {
	df.lastMessage = newLastMessage
}

func (df DuplicateFilter) IsDuplicate(body []byte) bool {
	return bytes.Equal(df.lastMessage, body)
}
