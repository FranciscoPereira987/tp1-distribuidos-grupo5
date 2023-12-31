package typing

import (
	"bytes"
	"encoding/binary"
	"io"
)

type FastestFilter struct {
	ID          [16]byte
	Origin      string
	Destination string
	Duration    uint32
	Stops       string
}

func (data *FastestFilter) Marshal(b *bytes.Buffer) {
	b.Write(data.ID[:])
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.Duration)
	WriteString(b, data.Stops)
}

func FastestFilterMarshal(b *bytes.Buffer, data *Flight) {
	b.Write(data.ID[:])
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.Duration)
	WriteString(b, data.Stops)
}

func FastestFilterUnmarshal(r *bytes.Reader) (data FastestFilter, err error) {
	_, err = io.ReadFull(r, data.ID[:])
	if err == nil {
		data.Origin, err = ReadString(r)
	}
	if err == nil {
		data.Destination, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Duration)
	}
	if err == nil {
		data.Stops, err = ReadString(r)
	}

	return data, err
}
