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
	stateMan.AddToState("recieved-queue", df.queueLast)
}

func (df *DuplicateFilter) recoverQueue(value any) {
	queue, ok := value.([]any)
	if ok {
		for _, element := range queue {
			if asString, ok := element.(string); ok {
				df.lastMessages[asString] = true
				df.queueLast = append(df.queueLast, asString)
			}
		}
	}
}

func (df *DuplicateFilter) RecoverFromState(stateMan *state.StateManager) {
	if value := stateMan.GetString("last-received"); value != "" {
		df.LastMessage = value
	}
	if value := stateMan.Get("recieved-queue"); value != nil {
		df.recoverQueue(value)
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
	h, err := typing.HeaderUnmarshal(r)

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
	h, err := typing.HeaderUnmarshal(r)
	if err == nil {
		dup = h.ID == df.LastMessage
	}
	return
}
