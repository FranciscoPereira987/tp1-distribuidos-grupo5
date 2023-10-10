package middleware

import (
	"bytes"
	"encoding/binary"
	"io"
)

func Q3Marshal(data StopsFilterData) []byte {
	buf := make([]byte, 1, 40)
	buf[0] = Query3Flag

	buf = append(buf, data.ID[:]...)

	buf = AppendString(buf, data.Origin)
	buf = AppendString(buf, data.Destination)

	buf = binary.LittleEndian.AppendUint32(buf, data.Duration)

	buf = AppendString(buf, data.Stops)

	return buf
}

type ResultQ3 struct {
	ID          [16]byte
	Origin      string
	Destination string
	Duration    uint32
	Stops       string
}

func ResultQ3Marshal(data ResultQ3) []byte {
	filter := StopsFilterData{
		ID: data.ID,
		Origin: data.Origin,
		Destination: data.Destination,
		Duration: data.Duration,
		Stops: data.Stops,
	}
	return Q3Marshal(filter)
}

func Q3Unmarshal(r *bytes.Reader) (data ResultQ3, err error) {
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
