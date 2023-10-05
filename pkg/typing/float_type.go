package typing

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/utils"
)

var (
	FLOAT_TYPE_NUMBER = byte(0x02)
	FLOAT_TYPE_LENGTH = 9
)

type FloatType struct {
	Value float64
}

func (f *FloatType) length() int {
	return FLOAT_TYPE_LENGTH
}

func (f *FloatType) Number() byte {
	return FLOAT_TYPE_NUMBER
}

func (f *FloatType) Serialize() []byte {
	bits := math.Float64bits(f.Value)
	return binary.BigEndian.AppendUint64(utils.GetHeader(f), bits)
}

func (f *FloatType) Trim(stream []byte) []byte {
	if err := utils.CheckHeader(f, stream); err != nil {
		return stream
	}
	return stream[f.length():]
}

func (f *FloatType) Deserialize(stream []byte) error {
	if err := utils.CheckHeader(f, stream); err != nil {
		return err
	}

	if err := CheckTypeLength(f.length(), stream); err != nil {
		return err
	}

	bits := binary.BigEndian.Uint64(stream[1:])
	f.Value = math.Float64frombits(bits)

	return nil
}

func (float *FloatType) AsRecord() []string {
	return []string{fmt.Sprint(float)}
}
