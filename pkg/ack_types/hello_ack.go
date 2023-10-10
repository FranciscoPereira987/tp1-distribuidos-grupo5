package ack_types

import (
	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	HELLOACK_TYPE_CODE = byte(0x03)
)

type HelloAckType struct {
}

func (data *HelloAckType) IsAckFrom(ack typing.Type) bool {
	_, ok := ack.(*HelloAckType)
	return ok
}

func (hello *HelloAckType) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(hello, stream); err != nil {
		return stream
	}
	return nil
}


func NewHelloAck() *HelloAckType {
	return &HelloAckType{
	}
}

func (helloAck *HelloAckType) Number() byte {
	return HELLOACK_TYPE_CODE
}

func (helloAck *HelloAckType) Serialize() []byte {
	
	return utils.GetHeader(helloAck)
}

func (helloAck *HelloAckType) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(helloAck, stream); err != nil {
		return err
	}
	
	return nil
}

func (ack *HelloAckType) AsRecord() []string {
	return []string{}
}
