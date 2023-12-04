package typing

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
)

var ErrUnsupported = errors.New("unsupported value")

type BatchHeader struct {
	WorkerId  string
	MessageId int64
}

func NewHeader(workerId string, messageId int64) BatchHeader {
	return BatchHeader{workerId, messageId}
}

func RecoverHeader(state map[string]any, workerId string) (h BatchHeader, err error) {
	h.WorkerId = workerId
	v, ok := state["message-id"]
	if !ok {
		return h, nil
	}
	num, ok := v.(json.Number)
	if !ok {
		return h, fmt.Errorf("%w: %#v", ErrUnsupported, v)
	}
	h.MessageId, err = num.Int64()
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
