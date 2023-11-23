package duplicates

import (
	"bytes"
	"context"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/middleware"
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

func NewDuplicateFilter(lastMessage []byte) DuplicateFilter {
	return DuplicateFilter{
		lastMessage: lastMessage,
	}
}

func (df *DuplicateFilter) ChangeLast(newLastMessage []byte) {
	df.lastMessage = newLastMessage
}

func (df DuplicateFilter) IsDuplicate(body []byte) bool {
	return bytes.Equal(df.lastMessage, body)
}
