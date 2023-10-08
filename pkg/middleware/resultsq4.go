package middleware

import (
	"bytes"
	"encoding/binary"
)

type ResultQ4 struct {
	Origin      string
	Destination string
	AvgPrice    float32
	MaxPrice    float32
}

func Q4Marshal(data ResultQ4) []byte {
	buf := make([]byte, 1, 20)
	buf[0] = Query4Flag

	buf = AppendString(buf, data.Origin)
	buf = AppendString(buf, data.Destination)

	var w bytes.Buffer
	_ = binary.Write(&w, binary.LittleEndian, data.AvgPrice)
	_ = binary.Write(&w, binary.LittleEndian, data.MaxPrice)
	buf = append(buf, w.Bytes()...)

	return buf
}

func Q4Unmarshal(r *bytes.Reader) (data ResultQ4, err error) {
	data.Origin, err = ReadString(r)
	if err == nil {
		data.Destination, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.AvgPrice))
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.MaxPrice))
	}

	return data, err
}
