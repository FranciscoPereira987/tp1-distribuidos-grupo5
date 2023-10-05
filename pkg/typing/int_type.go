package typing

import (
	"encoding/binary"
	"fmt"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	INT_TYPE_NUMBER = byte(0x00)
)

type IntType struct {
	Value uint32
}

func (intType IntType) length() int {
	return 5
}

func (intType *IntType) Number() byte {
	return INT_TYPE_NUMBER
}

func (intType *IntType) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(intType, stream); err != nil {
		return stream
	}
	return stream[intType.length():]
}

func (intType *IntType) Serialize() []byte {
	return binary.BigEndian.AppendUint32(utils.GetHeader(intType), intType.Value)
}

func (intType *IntType) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(intType, stream); err != nil {
		return err
	}
	if err := CheckTypeLength(intType.length(), stream); err != nil {
		return err
	}
	intType.Value = binary.BigEndian.Uint32(stream[1:])
	return nil
}

func (intType *IntType) AsRecord() []string {
	return []string{fmt.Sprint(intType.Value)}
}
