package proto

import "testing"

func TestParseLineEncoder(t *testing.T) {
	ev, err := ParseLine("E2:+3")
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if ev.Kind != EventEncoderDelta || ev.KnobID != 2 || ev.Delta != 3 {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParseLineButtonPress(t *testing.T) {
	ev, err := ParseLine("B5:1")
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if ev.Kind != EventButtonPress || ev.KnobID != 5 {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParseLineButtonReleaseRejected(t *testing.T) {
	if _, err := ParseLine("B5:0"); err == nil {
		t.Fatal("expected error for unsupported button value")
	}
}
