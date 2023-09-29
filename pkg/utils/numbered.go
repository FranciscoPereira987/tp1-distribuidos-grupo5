package utils

import "errors"

type Numbered interface {
	Number() byte
}

func CheckHeader(value Numbered, stream []byte) error {
	if value.Number() != stream[0] {
		return errors.New("not same Type")
	}

	return nil
}

func GetHeader(value Numbered) []byte {
	return []byte{value.Number()}
}
