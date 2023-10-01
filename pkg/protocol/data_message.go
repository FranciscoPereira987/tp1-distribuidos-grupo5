package protocol

import (
	"encoding/binary"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	DATA_OP_CODE = byte(0x03)
)

type DataMessage struct {
	data_type typing.Type
}

func NewDataMessage(value typing.Type) *DataMessage {
	return &DataMessage{
		value,
	}
}

func (data DataMessage) Type() typing.Type {
	return data.data_type
}

func (errMes *DataMessage) IsResponseFrom(message Message) bool {
	return false
}

func (hello *DataMessage) Number() byte {
	return DATA_OP_CODE
}

func (hello *DataMessage) Marshall() []byte {
	header := utils.GetHeader(hello)
	body := hello.data_type.Serialize()
	header = binary.BigEndian.AppendUint32(header, uint32(len(body)))
	return append(header, body...)
}

func (hello *DataMessage) UnMarshall(stream []byte) error {
	if err := utils.CheckHeader(hello, stream); err != nil {
		return err
	}

	type_size, err := CheckMessageLength(stream)
	if err != nil {
		return err
	}
	if err := typing.CheckTypeLength(int(type_size), stream[5:]); err != nil {
		return err
	}
	if err := hello.data_type.Deserialize(stream[5:]); err != nil {
		return err
	}
	return nil
}

func (hello *DataMessage) Response() Message {
	return NewDataAckMessage()
}
