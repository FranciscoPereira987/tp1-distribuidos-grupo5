package typing_test

import (
	"bytes"
	"encoding/binary"
	"math"
	"testing"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"
)

func TestFloatSerialization(t *testing.T) {
	value := typing.FloatType{
		299.0,
	}
	expected := binary.BigEndian.AppendUint64([]byte{value.Number()}, math.Float64bits(value.Value))
	result := value.Serialize()

	if !bytes.Equal(expected, result) {
		t.Fatalf("expected: %s, got: %s", expected, result)
	}
}

func TestFloatDeserialization(t *testing.T) {
	value := typing.FloatType{
		56.7893,
	}
	result := &typing.FloatType{}

	stream := value.Serialize()

	result.Deserialize(stream)

	if value.Value != result.Value {
		t.Fatalf("expected: %f, got: %f", value.Value, result.Value)
	}
}
