package typing

import "encoding/binary"

var (
	INT_TYPE_NUMBER = byte(0x00)
)

type IntType struct {
	Value uint32
}

func (intType IntType) length() int {
	return 5
}

func (intType *IntType) TypeNumber() byte {
	return INT_TYPE_NUMBER
}

func (intType *IntType) Serialize() []byte {
	return binary.BigEndian.AppendUint32(GetHeader(intType), intType.Value)
}

func (intType *IntType) Deserialize(stream []byte) error {
	if err := CheckHeader(intType, stream); err != nil {
		return err
	}
	if err := CheckTypeLength(intType.length(), stream); err != nil {
		return err
	}
	intType.Value = binary.BigEndian.Uint32(stream[1:])
	return nil
}
