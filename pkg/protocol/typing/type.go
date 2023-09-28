package typing

import "errors"

type Type interface {
	TypeNumber() byte
	Serialize() []byte
	Deserialize([]byte) error
}

func checkHeader(value Type, stream []byte) error {
	if value.TypeNumber() != stream[0] {
		return errors.New("not same Type")
	}

	return nil
}

func getHeader(value Type) []byte {
	return []byte{value.TypeNumber()}
}

func checkTypeLength(typeLength int, stream []byte) error {
	if typeLength != len(stream) {
		return errors.New("type length does not match")
	}

	return nil
}
