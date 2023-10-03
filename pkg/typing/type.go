package typing

import (
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

type Type interface {
	utils.Numbered
	Serialize() []byte
	Deserialize([]byte) error
	Trim([]byte) []byte
}

func CheckTypeLength(typeLength int, stream []byte) error {
	if typeLength != len(stream) {
		return errors.New("type length does not match")
	}

	return nil
}

func GetTypeFromStream(to Type, stream []byte) (value []byte, next []byte) {
	next = to.Trim(stream)
	value = stream[:len(stream)-len(next)]
	return
}