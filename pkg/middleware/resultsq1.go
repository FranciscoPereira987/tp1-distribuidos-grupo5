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
	buf = AppendString(buf, data.Destination)

	var w bytes.Buffer
	_ = binary.Write(&w, binary.LittleEndian, data.Price)
	buf = append(buf, w.Bytes()...)

	buf = AppendString(buf, data.Stops)

	return buf
}

type ResultQ1 struct {
	ID          [16]byte
	Origin      string
	Destination string
	Price       float32
	Stops       string
}

func ResultQ1marshal(data ResultQ1) []byte {
	filter := StopsFilterData{
		ID: data.ID,
		Origin: data.Origin,
		Destination: data.Destination,
		Price: data.Price,
		Stops: data.Stops,
	}
	return Q1Marshal(filter)
}

func Q1Unmarshal(r *bytes.Reader) (data ResultQ1, err error) {
	_, err = io.ReadFull(r, data.ID[:])
	if err == nil {
		data.Origin, err = ReadString(r)
	}
	if err == nil {
		data.Destination, err = ReadString(r)
	}
	if err == nil {
		err = binary.Read(r, binary.LittleEndian, &(data.Price))
	}
	if err == nil {
		data.Stops, err = ReadString(r)
	}

	return data, err
}
