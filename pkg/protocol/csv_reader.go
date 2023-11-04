package protocol

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
)

var ErrMissingFields = errors.New("missing required fields")

func NewCsvReader(in io.Reader, comma rune, fields []string) (*csv.Reader, []int, error) {
	var indices []int
	r := csv.NewReader(in)
	r.ReuseRecord = true
	r.Comma = comma

	record, err := r.Read()
	if err != nil {
		return nil, nil, err
	}
	for _, key := range fields {
		for i, field := range record {
			if field == key {
				indices = append(indices, i)
				break
			}
		}
	}
	if len(indices) != len(fields) {
		return nil, nil, fmt.Errorf("%w: have %v, need %v", ErrMissingFields, record, fields)
	}
	return r, indices, nil
}
