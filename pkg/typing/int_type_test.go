package typing_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

func TestIntSerialization(t *testing.T) {
	value := typing.IntType{
		999,
	}
	expected := binary.LittleEndian.AppendUint32([]byte{typing.INT_TYPE_NUMBER}, value.Value)

	result := value.Serialize()

	if !bytes.Equal(expected, result) {
		t.Fatalf("expected: %s, got: %s", expected, result)
	}
}

func TestIntDeserialization(t *testing.T) {
	value := typing.IntType{
		123456,
	}
	result := &typing.IntType{}

	stream := value.Serialize()

	result.Deserialize(stream)

	if value.Value != result.Value {
		t.Fatalf("expected: %d, got: %d", value.Value, result.Value)
	}
}
