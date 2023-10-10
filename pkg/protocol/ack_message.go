package protocol

import (
	"encoding/binary"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

const (
	ACK_OP_CODE = byte(0x02)
	TYPE_INDEX  = 5
)

type AckMessage struct {
}

func NewHelloAckMessage() Message {
	return &AckMessage{
	}
}


func (hello *AckMessage) Number() byte {
	return ACK_OP_CODE
}

func (hello *AckMessage) Marshal() []byte {
	header := utils.GetHeader(hello)
	header = binary.BigEndian.AppendUint32(header, 0)

	return header
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

	return nil
}

func (hello *AckMessage) Response() Message {
	return nil
}
