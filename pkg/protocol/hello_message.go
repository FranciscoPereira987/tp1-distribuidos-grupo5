package protocol

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	HELLO_OP_CODE = byte(0x01)
)

type HelloMessage struct{
	//The client_id should only be set by the server
	client_id uint32
}

func NewHelloMessage(client_id uint32) *HelloMessage{
	return &HelloMessage{
		client_id,
	}
}

func (hello *HelloMessage) Number() byte {
	return HELLO_OP_CODE
}

func (hello *HelloMessage) Marshall() []byte {
	return utils.GetHeader(hello)
}

func (hello *HelloMessage) UnMarshall(stream []byte) error {
	if err := utils.CheckHeader(hello, stream); err != nil {
		return err
	}
	if err := typing.CheckTypeLength(1, stream); err != nil {
		return err
	}
	return nil
}

func (hello *HelloMessage) Response() Message {
	return NewHelloAckMessage(hello.client_id)
}
