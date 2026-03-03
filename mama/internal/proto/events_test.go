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

func TestParseLineProtocolHello(t *testing.T) {
	ev, err := ParseLine("V:1")
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if ev.Kind != EventProtocolHello || ev.ProtocolVersion != 1 {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParseLineProtocolHelloInvalid(t *testing.T) {
	if _, err := ParseLine("V:0"); err == nil {
		t.Fatal("expected error for invalid protocol version")
	}
}

func TestIsProtocolCompatible(t *testing.T) {
	if !IsProtocolCompatible(HostProtocolVersion) {
		t.Fatal("expected host protocol version to be compatible")
	}
	if IsProtocolCompatible(HostProtocolVersion + 1) {
		t.Fatal("expected different version to be incompatible")
	}
}

func TestParseLineButtonReleaseRejected(t *testing.T) {
	if _, err := ParseLine("B5:0"); err == nil {
		t.Fatal("expected error for unsupported button value")
	}
}

func TestParseLineEncoderRelaxedFormatting(t *testing.T) {
	ev, err := ParseLine(" e2 : + 3 ")
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if ev.Kind != EventEncoderDelta || ev.KnobID != 2 || ev.Delta != 3 {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParseLineLegacyKEncoder(t *testing.T) {
	ev, err := ParseLine("K4:-2")
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if ev.Kind != EventEncoderDelta || ev.KnobID != 4 || ev.Delta != -2 {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParseLineLegacyKButtonPress(t *testing.T) {
	ev, err := ParseLine("k3:press")
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if ev.Kind != EventButtonPress || ev.KnobID != 3 {
		t.Fatalf("unexpected event: %+v", ev)
	}
}

func TestParseLineEmbeddedToken(t *testing.T) {
	ev, err := ParseLine("rx[serial]: E7:+1")
	if err != nil {
		t.Fatalf("ParseLine returned error: %v", err)
	}
	if ev.Kind != EventEncoderDelta || ev.KnobID != 7 || ev.Delta != 1 {
		t.Fatalf("unexpected event: %+v", ev)
	}
}
