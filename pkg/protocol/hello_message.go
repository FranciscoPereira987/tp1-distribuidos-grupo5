package protocol

import (
	"encoding/binary"
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	HELLO_OP_CODE = byte(0x01)
)

type HelloMessage struct {
	//The client_id should only be set by the server
	client_id uint32
}

func NewHelloMessage(client_id uint32) *HelloMessage {
	return &HelloMessage{
		client_id,
	}
}

func (hello *HelloMessage) IsResponseFrom(message Message) bool {
	return false
}

func (hello *HelloMessage) Number() byte {
	return HELLO_OP_CODE
}

func (hello *HelloMessage) Marshall() []byte {
	header := utils.GetHeader(hello)
	header = binary.BigEndian.AppendUint32(header, 0)
	return header
}

func (hello *HelloMessage) UnMarshall(stream []byte) error {
	if err := utils.CheckHeader(hello, stream); err != nil {
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

func (hello *HelloMessage) Response() Message {
	return NewHelloAckMessage(hello.client_id)
}
