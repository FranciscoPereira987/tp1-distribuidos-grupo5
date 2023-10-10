package protocol

import (
	"encoding/binary"
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

const (
	FIN_OP_CODE = byte(0xff)
)

type FinMessage struct {
}

func (fin *FinMessage) IsResponseFrom(message Message) bool {
	return false
}

func (fin *FinMessage) Number() byte {
	return FIN_OP_CODE
}

func (fin *FinMessage) Marshal() []byte {
	header := utils.GetHeader(fin)
	header = binary.BigEndian.AppendUint32(header, 0)
	return header
}

func (fin *FinMessage) UnMarshal(stream []byte) error {
	if err := utils.CheckHeader(fin, stream); err != nil {
		return err
	}
	body_length, err := CheckMessageLength(stream)
	if err != nil {
		return err
	}
	if body_length > 0 {
		return errors.New("invalid err message")
	}
	return nil
}

