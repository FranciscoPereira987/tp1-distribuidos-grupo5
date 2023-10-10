package protocol

import (
	"encoding/binary"
	"errors"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

const (
	DATA_OP_CODE = byte(0x03)
)

type Data interface {
	Message
	utils.Recorder
	Type() typing.Type
}

type DataMessage struct {
	data_type typing.Type
}

type MultiData struct {
	data map[byte]*DataMessage
	last *DataMessage
}

func NewMultiData() *MultiData {
	return &MultiData{
		data: make(map[byte]*DataMessage),
	}
}

func (multi *MultiData) Register(args ...*DataMessage) {
	for _, dataM := range args {
		multi.data[dataM.Type().Number()] = dataM
		multi.last = dataM
	}
}

func (multi *MultiData) IsResponseFrom(mess Message) bool {
	return multi.last.IsResponseFrom(mess)
}

func (multi *MultiData) Type() typing.Type {
	return multi.last.Type()
}

func (multi *MultiData) Number() byte {
	return DATA_OP_CODE
}

func (multi *MultiData) Marshal() []byte {
	return multi.last.Marshal()
}

func (multi *MultiData) UnMarshal(stream []byte) error {

	if len(stream) <= 5 {
		return errors.New("invalid data message")
	}
	key := stream[5]
	if _, ok := multi.data[key]; !ok {
		return errors.New("unknown data message")
	}

	multi.last = multi.data[key]

	return multi.last.UnMarshal(stream)
}


func (multi *MultiData) AsRecord() []string {
	return multi.last.AsRecord()
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

func (hello *DataMessage) Marshal() []byte {
	header := utils.GetHeader(hello)
	body := hello.data_type.Serialize()
	header = binary.BigEndian.AppendUint32(header, uint32(len(body)))
	return append(header, body...)
}

func (hello *DataMessage) UnMarshal(stream []byte) error {
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


func (data *DataMessage) AsRecord() []string {
	return data.Type().AsRecord()
}
