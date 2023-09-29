package protocol

import "github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"

var (
	DATA_OP_CODE = byte(0x03)
)

type DataMessage struct {
	data_type typing.Type
}

func (hello *DataMessage) Marshall() []byte {
	return nil
}

func (hello *DataMessage) UnMarshall(stream []byte) error {
	return nil
}

func (hello *DataMessage) Response() Message {
	return &DataMessage{}
}
