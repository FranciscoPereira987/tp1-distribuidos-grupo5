package protocol

import (
	"encoding/binary"
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

const (
	ERR_OP_CODE = byte(0x04)
)

type ErrMessage struct {
}

func (errMes *ErrMessage) IsResponseFrom(message Message) bool {
	return false
}

func (errMes *ErrMessage) Number() byte {
	return ERR_OP_CODE
}

func (errMes *ErrMessage) Marshal() []byte {
	header := utils.GetHeader(errMes)
	header = binary.LittleEndian.AppendUint32(header, 0)
	return header
}

func (errMes *ErrMessage) UnMarshal(stream []byte) error {
	if err := utils.CheckHeader(errMes, stream); err != nil {
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
