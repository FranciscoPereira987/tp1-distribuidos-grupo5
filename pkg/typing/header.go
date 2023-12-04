package typing

import (
	"bytes"
	"encoding/binary"
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/state"
)

type BatchHeader struct {
	WorkerId  string
	MessageId int64
}

func NewHeader(workerId string, messageId int64) BatchHeader {
	return BatchHeader{workerId, messageId}
}

func RecoverHeader(stateMan *state.StateManager, workerId string) (BatchHeader, error) {
	messageId, err := stateMan.GetInt64("message-id")
	h := NewHeader(workerId, messageId)
	if err != nil && errors.Is(err, state.ErrNotFound) {
		err = nil
	}
	return h, err
}

func (h BatchHeader) AddToState(state map[string]any) {
	state["message-id"] = h.MessageId
}

func (h *BatchHeader) Unmarshal(r *bytes.Reader) error {
	id, err := ReadString(r)
	if err == nil {
		h.WorkerId = id
		err = binary.Read(r, binary.LittleEndian, &h.MessageId)
	}
	return err
}

func (h BatchHeader) Marshal(b *bytes.Buffer) {
	WriteString(b, h.WorkerId)
	binary.Write(b, binary.LittleEndian, h.MessageId)
}
