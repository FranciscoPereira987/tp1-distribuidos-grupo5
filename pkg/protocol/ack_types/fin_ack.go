package ack_types

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	FINACK_TYPE_NUMBER = byte(0x04)
)

type FinAckType struct{}

func (fin *FinAckType) Number() byte {
	return FINACK_TYPE_NUMBER
}

func (fin *FinAckType) Serialize() []byte {
	return utils.GetHeader(fin)
}

func (fin *FinAckType) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(fin, stream); err != nil {
		return err
	}
	return nil
}
