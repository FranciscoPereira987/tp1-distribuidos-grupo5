package protocol

import (
	"encoding/binary"
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	DATA_OP_CODE = byte(0x03)
)

type DataMessage struct {
	data_type typing.Type
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
	if len(stream) < 5 {
		return errors.New("message too short")
	}
	type_size := binary.BigEndian.Uint32(stream[1:5])
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
