package middleware

import (
	"encoding/binary"
	"io"
)

func Q3Marshal(data StopsFilterData) []byte {
	buf := make([]byte, 1, 40)
	buf[0] = mid.Query3Flag

	buf = append(buf, data.ID[:]...)

	buf = AppendString(buf, data.Origin)
	buf = AppendString(buf, data.Destiny)

	buf = binary.LittleEndian.AppendUint32(buf, data.Duration)

	buf = AppendString(buf, data.Stops)

	return buf
}

type Q3Result struct {
	ID       [16]byte
	Origin   string
	Destiny  string
	Duration uint32
	Stops    string
}

func Q3Unmarshal(r io.Reader) (data ResultQ3, err error) {
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
		data.Stops, err = ReadString(r)
	}

	return data, err
}
