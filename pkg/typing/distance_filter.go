package typing

import (
	"bytes"
	"encoding/binary"
	"io"
)

type DistanceFilter struct {
	ID          [16]byte
	Origin      string
	Destination string
	Distance    uint32
}

func DistanceFilterMarshal(b *bytes.Buffer, data *Flight) {
	b.Write(data.ID[:])
	WriteString(b, data.Origin)
	WriteString(b, data.Destination)
	binary.Write(b, binary.LittleEndian, data.Distance)
}

func DistanceFilterUnmarshal(buf []byte) (data DistanceFilter, err error) {
	r := bytes.NewReader(buf)

	_, err = io.ReadFull(r, data.ID[:])

	if err == nil {
		data.Origin, err = ReadString(r)
	}
	if err == nil {
		data.Destination, err = ReadString(r)
	}

	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &data.Distance)
	}

	return data, err
}
