package typing

import (
	"bytes"
)

type BatchHeader struct {
	WorkerId  string
	MessageId string
}

func NewHeader(workerId string, messageId string) BatchHeader {
	return BatchHeader{workerId, messageId}
}

func (h *BatchHeader) Unmarshal(r *bytes.Reader) error {
	id, err := ReadString(r)
	if err == nil {
		h.WorkerId = id
		h.MessageId, err = ReadString(r)
	}
	return err
}

func (h BatchHeader) Marshal(b *bytes.Buffer) {
	WriteString(b, h.WorkerId)
	WriteString(b, h.MessageId)
}
