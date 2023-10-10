package typing_test

import (
	"testing"

	"github.com/franciscopereira987/tp1-distribuidos/pkg/typing"
)

func TestDurationParse(t *testing.T) {
	duration := "PT6H55M"

	value, err := typing.ParseDuration(duration)
	if err != nil {
		t.Fatalf("failed with error: %s", err)
	}
	if value != (55 + 6*60) {
		t.Fatalf("expected: %d, got: %d", 55+6*60, value)
	}
}
