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
func ResultUnmarshal(r *bytes.Reader) ([]string, error) {
	flag, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("Error reading flight data: %w", io.ErrUnexpectedEOF)
	}

	switch flag {
	case Query1Flag:
		return ResultQ1Unmarshal(r)
	case Query2Flag:
		return ResultQ2Unmarshal(r)
	case Query3Flag:
		return ResultQ3Unmarshal(r)
	case Query4Flag:
		return ResultQ4Unmarshal(r)
	default:
		return nil, fmt.Errorf("Unknown format specifier: %d", flag)
	}
}
