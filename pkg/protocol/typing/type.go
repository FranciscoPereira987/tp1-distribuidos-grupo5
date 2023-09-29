package typing

import "errors"

type Type interface {
	TypeNumber() byte
	Serialize() []byte
	Deserialize([]byte) error
}

func CheckHeader(value Type, stream []byte) error {
	if value.TypeNumber() != stream[0] {
		return errors.New("not same Type")
	}

	return nil
}

func GetHeader(value Type) []byte {
	return []byte{value.TypeNumber()}
}

func CheckTypeLength(typeLength int, stream []byte) error {
	if typeLength != len(stream) {
		return errors.New("type length does not match")
	}

	return nil
}
