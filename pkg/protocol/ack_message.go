package protocol

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/ack_types"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"
)

var (
	ACK_OP_CODE = byte(0x02)
)

type AckMessage struct {
	/*
		The body of an Ack message contains information about:

			1. The Ack Type
			2. Extra information the Ack needs
	*/
	ack_body typing.Type
}

func NewHelloAckMessage(client_id uint32) Message {
	return &AckMessage{
		ack_body: ack_types.NewHelloAck(client_id),
	}
}

func NewDataAckMessage() Message {
	return &AckMessage{
		ack_body: &ack_types.DataAckType{},
	}
}

func NewFinAckMessage() Message {
	return &AckMessage{
		ack_body: &ack_types.FinAckType{},
	}
}

func (hello *AckMessage) Marshall() []byte {
	return nil
}

func (hello *AckMessage) UnMarshall(stream []byte) error {
	return nil
}

func (hello *AckMessage) Response() Message {
	return &AckMessage{}
}
