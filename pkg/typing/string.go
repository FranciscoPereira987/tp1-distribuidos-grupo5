package typing

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

type stringReader interface {
	io.ByteReader
	io.Reader
}

var ErrLength = errors.New("string too long")

func WriteString(b *bytes.Buffer, s string) error {
	if len(s) > 255 {
		return fmt.Errorf("%w: len=%d", ErrLength, len(s))
	}

	b.WriteByte(byte(len(s)))
	b.WriteString(s)

	return nil
}

func ReadString(r stringReader) (string, error) {
	n, err := r.ReadByte()
	if err != nil {
		return "", err
	}

	buf := make([]byte, n)
	_, err = io.ReadFull(r, buf)

	return string(buf), err
}
