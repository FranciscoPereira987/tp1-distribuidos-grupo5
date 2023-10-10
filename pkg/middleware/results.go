package middleware

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
)

const (
	FlightType = iota
	Query1Flag
	Query2Flag
	Query3Flag
	Query4Flag
	CoordFlag
)

// Get the result with a type switch
func ResultUnmarshal(buf []byte) (any, error) {
	if len(buf) < 1 {
		return nil, fmt.Errorf("Error reading flight data: %w", io.ErrUnexpectedEOF)
	}

	r := bytes.NewReader(buf[1:])

	switch buf[0] {
	case Query1Flag:
		return Q1Unmarshal(r)
	// case Query2Flag:
	// 	return Q2Unmarshal(r)
	case Query3Flag:
		return Q3Unmarshal(r)
	// case Query4Flag:
	// 	return Q4Unmarshal(r)
	default:
		return nil, fmt.Errorf("Unknown format specifier: %d", buf[0])
	}
}

func AppendString(buf []byte, s string) []byte {
	buf = binary.AppendUvarint(buf, uint64(len(s)))
	return append(buf, s...)
}

func ReadString(r *bytes.Reader) (string, error) {
	n, err := binary.ReadUvarint(r)
	if err != nil {
		return "", err
	}

	buf := make([]byte, n)
	_, err = io.ReadFull(r, buf)

	return string(buf), err
}
