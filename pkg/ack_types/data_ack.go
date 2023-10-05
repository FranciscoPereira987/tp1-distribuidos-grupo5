package ack_types

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	DATAACK_TYPE_NUMBER = byte(0x05)
)

type DataAckType struct {
}

func (data *DataAckType) IsAckFrom(ack typing.Type) bool {
	_, ok := ack.(*DataAckType)
	return ok
}

func (data *DataAckType) Number() byte {
	return DATAACK_TYPE_NUMBER
}

func (data *DataAckType) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(data, stream); err != nil {
		return stream
	}
	return stream[1:]
}

func (data *DataAckType) Serialize() []byte {
	return utils.GetHeader(data)
}

func (data *DataAckType) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(data, stream); err != nil {
		return err
	}

	return nil
}

func (data *DataAckType) AsRecord() []string {
	return []string{}
}
