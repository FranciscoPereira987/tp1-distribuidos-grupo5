package middleware

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	Query1Flag = iota
	Query2Flag
	Query3Flag
	Query4Flag
)

// Get the result with a type switch
func ResultUnmarshal(buf []byte) (any, error) {
	if len(buf) < 1 {
		return data, fmt.Errorf("Error reading flight data: %w", io.ErrUnexpectedEOF)
	}

	r := bytes.NewReader(buf[1:])

	switch buf[0] {
	case Query1Flag:
		return UnmarshalQ1(r)
	case Query2Flag:
		return UnmarshalQ2(r)
	case Query3Flag:
		return UnmarshalQ3(r)
	case Query4Flag:
		return UnmarshalQ4(r)
	default:
		return nil, fmt.Errorf("Unknown format specifier: %d", buf[0])
	}
}

type ResultQ1 struct {
	ID       [16]byte
	Origin   string
	Destiny  string
	Price    float32
	Stops    string
}

func UnmarshalQ1(r io.Reader) (data ResultQ1, err error) {
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

func AppendString(buf []byte, s string) []byte {
	buf = binary.AppendUvarint(buf, len(s))
	return append(buf, s...)
}

func ReadString(r io.Reader) (string, error) {
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return "", err
	}

	buf := make([]byte, n)
	_, err = io.ReadFull(r, buf)

	return string(buf), err
}
