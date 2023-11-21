package duplicates

import (
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

func NewDuplicateFilter(lastMessage []byte) *DuplicateFilter {
	if lastMessage == nil {
		lastMessage = make([]byte, 0)
	}
	return &DuplicateFilter{
		lastMessage: lastMessage,
	}
}

func (df *DuplicateFilter) ChangeLast(new []byte) {
	df.lastMessage = new
}

func (df *DuplicateFilter) IsDuplicate(body []byte) (dup bool) {
	dup = len(body) == len(df.lastMessage)
	for i := 0; dup && i < len(body); i++ {
		dup = body[i] == df.lastMessage[i]
	}
	return
}
