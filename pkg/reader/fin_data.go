package reader

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	DATA_FIN_TYPE_NUMBER = byte(0x07)
)

type DataFin struct{
	processed *typing.IntType
}

func (data *DataFin) Number() byte {
	return DATA_FIN_TYPE_NUMBER
}

func FinData(processed uint32) *DataFin {
	return &DataFin{
		processed: &typing.IntType{Value: processed},
	}
}

func (data *DataFin) GetProcessed() uint32 {
	return data.processed.Value
}

func (data *DataFin) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(data, stream); err != nil {
		return stream
	}
	return stream[1:]
}

func (data *DataFin) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(data, stream); err != nil {
		return err
	}
	if err := data.processed.Deserialize(stream[1:]); err != nil {
		return err
	}
	return nil
}

func (data *DataFin) Serialize() []byte {
	return append(utils.GetHeader(data), data.processed.Serialize()...)
}

func (Data *DataFin) AsRecord() []string {
	return []string{}
}
