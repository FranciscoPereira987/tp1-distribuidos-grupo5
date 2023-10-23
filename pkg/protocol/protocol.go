package protocol

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

var ErrNotQueryResult = errors.New("not a query result")

var ResultEOF = []string{"0"}

func SplitRecord(record []byte) (int, []byte, error) {
	if len(record) == 0 {
		return 0, nil, io.ErrShortBuffer
	}
	head, tail, _ := bytes.Cut(record, []byte(","))
	if len(head) != 1 {
		return 0, nil, fmt.Errorf("%w: tag=%q", ErrNotQueryResult, head)
	}

	tag := int(head[0] - '0')
	if tag < 0 || tag > 4 {
		return 0, nil, fmt.Errorf("%w: tag=%d", ErrNotQueryResult, tag)
	}
	if tag == 0 {
		return 0, nil, io.EOF
	}

	return tag, tail, nil
}
