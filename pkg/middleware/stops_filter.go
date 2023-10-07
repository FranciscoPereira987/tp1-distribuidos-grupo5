package middleware

import (
	"bytes"
	"encoding/binary"
	"io"
)

type StopsFilterData struct {
	ID       [16]byte
	Origin   string
	Destiny  string
	Duration uint32
	Price    float32
	Stops    string
}

func StopsFilterUnmarshal(buf []byte) (data StopsFilterData, err error) {
	r := bytes.NewReader(buf)
	_, err = io.ReadFull(r, data.ID[:])
	if err == nil {
		data.Origin, err = ReadString(r)
	}
	if err == nil {
		data.Destiny, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Duration))
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Price))
	}
	if err == nil {
		data.Stops, err = ReadString(r)
	}

	return data, err
}
