package middleware

import (
	"bytes"
	"encoding/binary"
	"io"
)

type FastestFilterData ResultQ3

func FastestFilterUnmarshal(buf []byte) (data FastestFilterData, err error) {
	r := bytes.NewReader(buf)
	_, err = io.ReadFull(r, data.ID[:])
	if err == nil {
		data.Origin, err = ReadString(r)
	}
	if err == nil {
		data.Destination, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Duration))
	}
	if err == nil {
		data.Stops, err = ReadString(r)
	}

	return data, err
}
