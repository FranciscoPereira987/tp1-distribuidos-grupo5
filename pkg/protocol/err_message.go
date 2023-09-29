package protocol

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	ERR_OP_CODE = byte(0x04)
)

type ErrMessage struct {
}

func (errMes *ErrMessage) Number() byte {
	return ERR_OP_CODE
}

func (errMes *ErrMessage) Marshall() []byte {
	return utils.GetHeader(errMes)
}

func (errMes *ErrMessage) UnMarshall(stream []byte) error {
	if err := utils.CheckHeader(errMes, stream); err != nil {
		return err
	}
	if err := typing.CheckTypeLength(1, stream); err != nil {
		return err
	}
	return nil
}

func (errMes *ErrMessage) Response() Message {
	return NewFinAckMessage()
}
