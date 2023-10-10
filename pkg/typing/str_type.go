package typing

import (
	"encoding/binary"
	"errors"
	"math"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	STR_TYPE_NUMBER = byte(0x01)
)

type StrType struct {
	value string
}

func New() *StrType {
	return &StrType{""}
}

func NewStr(str string) (*StrType, error) {
	if len(str) > math.MaxInt16 {
		return nil, errors.New("string too large")
	}

	return &StrType{
		str,
	}, nil

}

func (str StrType) Value() string {
	return str.value
}

func (str StrType) length() int {
	return len(str.value)
}

func (str StrType) getValueLength(stream []byte) (int, error) {
	if len(stream) < 3 {
		return 0, errors.New("stream too short")
	}

	return int(binary.BigEndian.Uint16(stream[1:])), nil
}

func (str *StrType) deserializeValue(stream []byte, strLength int) error {

	if err := CheckTypeLength(strLength, stream); err != nil {
		return err
	}

	str.value = string(stream)
	return nil
}

func (str *StrType) Number() byte {
	return STR_TYPE_NUMBER
}

func (str *StrType) Serialize() []byte {
	header := utils.GetHeader(str)
	header = binary.BigEndian.AppendUint16(header, uint16(str.length()))

	return append(header, []byte(str.value)...)
}

func (str *StrType) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(str, stream); err != nil {
		return stream
	}
	strLength, err := str.getValueLength(stream)
	if err != nil {
		return stream
	}
	return stream[strLength+3:]
}

func (str *StrType) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(str, stream); err != nil {
		return err
	}
	strLength, err := str.getValueLength(stream)
	if err != nil {
		return err
	}
	return str.deserializeValue(stream[3:], strLength)
}

func (str *StrType) AsRecord() []string {
	return []string{str.value}
}
