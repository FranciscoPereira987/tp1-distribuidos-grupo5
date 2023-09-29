package ack_types

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	HELLOACK_TYPE_CODE = byte(0x03)
)

type HelloAckType struct {
	clientId typing.IntType
}

func (hello *HelloAckType) ClientId() uint32 {
	return hello.clientId.Value
}

func NewHelloAck(value uint32) *HelloAckType {
	return &HelloAckType{
		clientId: typing.IntType{
			Value: value,
		},
	}
}

func (helloAck *HelloAckType) Number() byte {
	return HELLOACK_TYPE_CODE
}

func (helloAck *HelloAckType) Serialize() []byte {
	header := utils.GetHeader(helloAck)
	body := helloAck.clientId.Serialize()
	return append(header, body...)
}

func (helloAck *HelloAckType) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(helloAck, stream); err != nil {
		return err
	}
	if err := helloAck.clientId.Deserialize(stream[1:]); err != nil {
		return err
	}
	return nil
}
