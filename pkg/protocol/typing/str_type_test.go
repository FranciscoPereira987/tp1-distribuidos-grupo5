package typing_test

import (
	"bytes"
	"testing"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/protocol/typing"
)

func TestStrSerialization(t *testing.T) {
	value, err := typing.NewStr("Hello world")
	if err != nil {
		t.Fatalf("got error: %s", err)
	}
	expected := []byte("Hello world")
	result := value.Serialize()

	if !bytes.Equal(expected, result[3:]) {
		t.Fatalf("expected: %s, got: %s", expected, result[3:])
	}
}

func TestStrDeserialization(t *testing.T) {
	value, err := typing.NewStr("I'm a nice message")

	if err != nil {
		t.Fatalf("got error: %s", err)
	}

	result, err := typing.NewStr("")

	if err != nil {
		t.Fatalf("got error: %s", err)
	}

	stream := value.Serialize()

	result.Deserialize(stream)

	if value.Value() != result.Value() {
		t.Fatalf("expected: %s, got: %s", value.Value(), result.Value())
	}
}
