package typing

import (
	"bytes"
	"encoding/hex"
)

type BatchHeader struct {
	ID string
}

func HeaderMarshal(b *bytes.Buffer, data *Flight) {
	h := BatchHeader{
		ID: hex.EncodeToString(data.ID[:]),
	}
	h.Marshal(b)
}

func HeaderIntoBuffer(b *bytes.Buffer, id string) {
	h := BatchHeader{
		ID: id,
	}
	h.Marshal(b)
}

func HeaderUnmarshal(r *bytes.Reader) (h *BatchHeader, err error) {
	h = new(BatchHeader)
	h.ID, err = ReadString(r)
	return
}

func (h BatchHeader) Marshal(b *bytes.Buffer) {
	WriteString(b, h.ID)
}
