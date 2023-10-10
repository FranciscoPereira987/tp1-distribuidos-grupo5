package protocol

import (
	"encoding/binary"
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

type Message interface {
	utils.Numbered
	Marshal() []byte
	UnMarshal([]byte) error
}

func CheckMessageLength(stream []byte) (int, error) {
	if len(stream) < 5 {
		return 0, errors.New("invalid message")
	}
	length := int(binary.BigEndian.Uint32(stream[1:5]))
	return length, nil
}
