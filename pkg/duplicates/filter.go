package duplicates

import (
	"bytes"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/sirupsen/logrus"
)

type DuplicateFilter struct {
	lastMessages map[string]string
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
	value, ok := stateMan.State["last-received"].(map[string]any)
	if ok {
		for key, val := range value {
			df.lastMessages[key] = val.(string)
		}
		logrus.Infof("Recovered from state: %s", df.lastMessages)
	} else {
		logrus.Info("Could not recover duplicates from state")
	}
}

func (df *DuplicateFilter) Update(r *bytes.Reader) (dup bool, err error) {
	var h typing.BatchHeader
	if err = h.Unmarshal(r); err == nil {
		if lastId, ok := df.lastMessages[h.WorkerId]; ok {
			dup = lastId == string(h.MessageId)
		}
		df.lastMessages[h.WorkerId] = string(h.MessageId)
	}
	return dup, err
}
