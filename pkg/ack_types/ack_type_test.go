package ack_types_test

import (
	"testing"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/ack_types"
)

func TestAckDeserialization(t *testing.T) {
	value := ack_types.NewHelloAck(56)
	result := ack_types.NewHelloAck(0)

	stream := value.Serialize()

	if err := result.Deserialize(stream); err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if result.ClientId() != value.ClientId() {
		t.Fatalf("expected: %d, got: %d", result.ClientId(), value.ClientId())
	}
}
