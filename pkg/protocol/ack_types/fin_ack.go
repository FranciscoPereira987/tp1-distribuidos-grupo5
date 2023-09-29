package ack_types

import "github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"

var (
	FINACK_TYPE_NUMBER = byte(0x04)
)

type FinAckType struct{}

func (fin *FinAckType) TypeNumber() byte {
	return FINACK_TYPE_NUMBER
}

func (fin *FinAckType) Serialize() []byte {
	return typing.GetHeader(fin)
}

func (fin *FinAckType) Deserialize(stream []byte) error {
	if err := typing.CheckHeader(fin, stream); err != nil {
		return err
	}
	return nil
}
