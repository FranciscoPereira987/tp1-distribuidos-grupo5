package typing

import (
	"bytes"
	"fmt"
	"io"
)

const (
	_ = iota
	Query1Flag
	Query2Flag
	Query3Flag
	Query4Flag
)

// returns the result as a record (a slice of fields)
func ResultUnmarshal(buf []byte) ([]string, error) {
	if len(buf) < 1 {
		return nil, fmt.Errorf("Error reading flight data: %w", io.ErrUnexpectedEOF)
	}

	r := bytes.NewReader(buf[1:])

	switch buf[0] {
	case Query1Flag:
		return ResultQ1Unmarshal(r)
	case Query2Flag:
		return ResultQ2Unmarshal(r)
	case Query3Flag:
		return ResultQ3Unmarshal(r)
	case Query4Flag:
		return ResultQ4Unmarshal(r)
	default:
		return nil, fmt.Errorf("Unknown format specifier: %d", buf[0])
	}
}
