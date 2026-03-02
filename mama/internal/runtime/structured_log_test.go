package runtime

import (
	"errors"
	"testing"
)

func TestStructuredMessageDeterministicOrder(t *testing.T) {
	msg := StructuredMessage("serial_state", Fields{
		"baud":  115200,
		"state": "connecting",
		"port":  "COM3",
	})

	const want = `event=serial_state baud=115200 port="COM3" state="connecting"`
	if msg != want {
		t.Fatalf("StructuredMessage() = %q, want %q", msg, want)
	}
}

func TestStructuredMessageFormatsErrors(t *testing.T) {
	msg := StructuredMessage("serial_state", Fields{
		"err":   errors.New("access denied"),
		"state": "reconnecting",
	})

	const want = `event=serial_state err="access denied" state="reconnecting"`
	if msg != want {
		t.Fatalf("StructuredMessage() = %q, want %q", msg, want)
	}
}
