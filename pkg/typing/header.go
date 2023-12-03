package typing

import (
	"bytes"
	"encoding/binary"
)

type BatchHeader struct {
	WorkerId  string
	MessageId uint64
}

func NewHeader(workerId string, messageId uint64) BatchHeader {
	return BatchHeader{workerId, messageId}
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
