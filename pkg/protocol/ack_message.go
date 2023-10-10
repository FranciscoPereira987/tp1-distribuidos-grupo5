package protocol

import (
	"encoding/binary"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/ack_types"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

const (
	ACK_OP_CODE = byte(0x02)
	TYPE_INDEX  = 5
)

type AckMessage struct {
	/*
		The body of an Ack message contains information about:

			1. The Ack Type
			2. Extra information the Ack needs
	*/
	ack_body ack_types.AckType
}

func (ack AckMessage) bodyNumber() byte {
	return ack.ack_body.Number()
}

func NewHelloAckMessage() Message {
	return &AckMessage{
		ack_body: &ack_types.HelloAckType{},
	}
}


func (hello *AckMessage) Number() byte {
	return ACK_OP_CODE
}

func (hello *AckMessage) Marshal() []byte {
	header := utils.GetHeader(hello)
	body := hello.ack_body.Serialize()
	header = binary.BigEndian.AppendUint32(header, uint32(len(body)))

	return append(header, body...)
}

func (hello *AckMessage) UnMarshal(stream []byte) error {

	if err := utils.CheckHeader(hello, stream); err != nil {
		return err
	}
	body_length, err := CheckMessageLength(stream)
	if err != nil {
		return err
	}
	if err := typing.CheckTypeLength(body_length, stream[5:]); err != nil {
		return err
	}

	return hello.ack_body.Deserialize(stream[5:])
}

func (hello *AckMessage) Response() Message {
	return nil
}
