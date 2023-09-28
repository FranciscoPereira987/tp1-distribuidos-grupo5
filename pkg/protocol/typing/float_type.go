package typing

import (
	"encoding/binary"
	"math"
)

var (
	FLOAT_TYPE_NUMBER = byte(0x02)
)

type FloatType struct {
	Value float64
}

func (f *FloatType) length() int {
	return 9
}

func (f *FloatType) TypeNumber() byte {
	return FLOAT_TYPE_NUMBER
}

func (f *FloatType) Serialize() []byte {
	bits := math.Float64bits(f.Value)
	return binary.BigEndian.AppendUint64(getHeader(f), bits)
}

func (f *FloatType) Deserialize(stream []byte) error {
	if err := checkHeader(f, stream); err != nil {
		return err
	}

	if err := checkTypeLength(f.length(), stream); err != nil {
		return err
	}

	bits := binary.BigEndian.Uint64(stream[1:])
	f.Value = math.Float64frombits(bits)

	return nil
}
