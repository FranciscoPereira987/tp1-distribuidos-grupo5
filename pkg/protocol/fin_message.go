package protocol

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	FIN_OP_CODE = byte(0xff)
)

type FinMessage struct {
}

func (fin *FinMessage) Number() byte {
	return FIN_OP_CODE
}

func (fin *FinMessage) Marshall() []byte {
	return utils.GetHeader(fin)
}

func (fin *FinMessage) UnMarshall(stream []byte) error {
	if err := utils.CheckHeader(fin, stream); err != nil {
		return err
	}
	if err := typing.CheckTypeLength(1, stream); err != nil {
		return err
	}
	return nil
}

func (fin *FinMessage) Response() Message {
	return NewFinAckMessage()
}
