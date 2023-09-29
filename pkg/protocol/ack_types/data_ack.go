package ack_types

import "github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"

var (
	DATAACK_TYPE_NUMBER = byte(0x05)
)

type DataAckType struct {
}

func (data *DataAckType) TypeNumber() byte {
	return DATAACK_TYPE_NUMBER
}

func (data *DataAckType) Serialize() []byte {
	return typing.GetHeader(data)
}

func (data *DataAckType) Deserialize(stream []byte) error {
	if err := typing.CheckHeader(data, stream); err != nil {
		return err
	}

	return nil
}
