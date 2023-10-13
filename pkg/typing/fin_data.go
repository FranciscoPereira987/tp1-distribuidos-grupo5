package typing

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	DATA_FIN_TYPE_NUMBER = byte(0x07)
)

type DataFin struct {
}

func (data *DataFin) Number() byte {
	return DATA_FIN_TYPE_NUMBER
}

func FinData() *DataFin {
	return &DataFin{}
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
	if err := CheckTypeLength(1, stream); err != nil {
		return err
	}
	return nil
}

func (data *DataFin) Serialize() []byte {
	return utils.GetHeader(data)
}

func (Data *DataFin) AsRecord() []string {
	return []string{}
}
