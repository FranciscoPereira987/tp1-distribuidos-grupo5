package middleware

import (
	"bytes"
	"encoding/binary"
	"io"
)

func Q2Marshal(data DataQ2) []byte {
	buf := make([]byte, 1, 40)
	buf[0] = Query2Flag

	buf = append(buf, data.ID[:]...)

	buf = AppendString(buf, data.Origin)
	buf = AppendString(buf, data.Destination)

	var w bytes.Buffer
	_ = binary.Write(&w, binary.LittleEndian, data.TotalDistance)
	buf = append(buf, w.Bytes()...)

	return buf
}

type ResultQ2 DataQ2

type DataQ2 struct {
	ID          [32]byte
	Origin      string
	Destination string

	TotalDistance uint32
}

func Q2Unmarshal(r *bytes.Reader) (data DataQ2, err error) {

	_, err = io.ReadFull(r, data.ID[:])

	if err == nil {
		data.Origin, err = ReadString(r)
	}
	if err == nil {
		data.Destination, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.TotalDistance))
	}

	return data, err
}
