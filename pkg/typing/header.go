package typing

import (
	"bytes"
	"encoding/hex"
)

type BatchHeader struct {
	ID string
}

func HeaderMarshall(b *bytes.Buffer, data *Flight) {
	h := BatchHeader{
		ID: hex.EncodeToString(data.ID[:]),
	}
	h.Marshall(b)
}

func HeaderIntoBuffer(b *bytes.Buffer, id string) {
	h := BatchHeader{
		ID: id,
	}
	h.Marshall(b)
}

func HeaderUnmarshall(r *bytes.Reader) (h *BatchHeader, err error) {
	h = new(BatchHeader)
	h.ID, err = ReadString(r)
	return
}

func (h BatchHeader) Marshall(b *bytes.Buffer) {
	WriteString(b, h.ID)
}
