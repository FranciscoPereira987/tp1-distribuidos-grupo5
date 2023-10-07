package middleware

import (
	"bytes"
	"encoding/binary"
	"io"
)

func Q1Marshal(data StopsFilterData) []byte {
	buf := make([]byte, 1, 40)
	buf[0] = Query1Flag

	buf = append(buf, data.ID[:]...)

	buf = AppendString(buf, data.Origin)
	buf = AppendString(buf, data.Destiny)

	var w bytes.Buffer
	_ = binary.Write(w, binary.LittleEndian, data.Price)
	buf = append(buf, w.Bytes()...)

	buf = AppendString(buf, data.Stops)

	return buf
}

type ResultQ1 struct {
	ID      [16]byte
	Origin  string
	Destiny string
	Price   float32
	Stops   string
}

func Q1Unmarshal(r io.Reader) (data ResultQ1, err error) {
	_, err = io.ReadFull(r, data.ID[:])
	if err == nil {
		data.Origin, err = ReadString(r)
	}
	if err == nil {
		data.Destiny, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Price))
	}
	if err == nil {
		data.Stops, err = ReadString(r)
	}

	return data, err
}
