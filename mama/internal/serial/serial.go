package serialx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.bug.st/serial"
	"mama/internal/proto"
)

type Reader struct {
	port  lineReaderPort
	sleep func(time.Duration)
}

type lineReaderPort interface {
	Read(p []byte) (n int, err error)
	Close() error
}

const (
	maxLineBytes                = 1024 * 1024
	readChunkSize               = 256
	idleReadDelay               = 5 * time.Millisecond
	defaultProbeProtocolTimeout = 2500 * time.Millisecond
	probeReadTimeout            = 80 * time.Millisecond
	probeInitialWindow          = 450 * time.Millisecond
	probeDTRLowDuration         = 120 * time.Millisecond
	probeResetWait              = 1400 * time.Millisecond
)

var (
	mamaHelloLinePattern  = regexp.MustCompile(`(?i)^mama\s*[:=]\s*(?:hello|protocol|v)\s*[:=]\s*([0-9]+)\s*$`)
	mamaHelloTokenPattern = regexp.MustCompile(`(?i)\bmama\s*[:=]\s*(?:hello|protocol|v)\s*[:=]\s*([0-9]+)\b`)
)

func ListPorts() ([]string, error) {
	return serial.GetPortsList()
}

func Open(portName string, baud int) (*Reader, error) {
	mode := &serial.Mode{BaudRate: baud}
	p, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}
	if err := p.SetReadTimeout(serial.NoTimeout); err != nil {
		_ = p.Close()
		return nil, err
	}
	// Some boards reset on open; short grace helps.
	time.Sleep(1200 * time.Millisecond)

	return &Reader{port: p, sleep: time.Sleep}, nil
}

func Probe(portName string, baud int) error {
	mode := &serial.Mode{BaudRate: baud}
	p, err := serial.Open(portName, mode)
	if err != nil {
		return err
	}
	return p.Close()
}

// ProbeProtocolHello opens a serial port and waits for a protocol hello line (MAMA:HELLO:<n>).
// It returns the firmware protocol version when compatible with this host.
func ProbeProtocolHello(portName string, baud int, timeout time.Duration) (int, error) {
	portName = strings.TrimSpace(portName)
	if portName == "" {
		return 0, fmt.Errorf("port is required")
	}
	if baud <= 0 {
		return 0, fmt.Errorf("baud must be > 0")
	}
	if timeout <= 0 {
		timeout = defaultProbeProtocolTimeout
	}

	mode := &serial.Mode{BaudRate: baud}
	p, err := serial.Open(portName, mode)
	if err != nil {
		return 0, err
	}
	defer func() { _ = p.Close() }()

	if err := p.SetReadTimeout(probeReadTimeout); err != nil {
		return 0, err
	}

	version, err := probeProtocolHelloWithWake(p, timeout)
	if err != nil {
		return 0, err
	}
	if !proto.IsProtocolCompatible(version) {
		return 0, fmt.Errorf("protocol mismatch: firmware=%d host=%d", version, proto.HostProtocolVersion)
	}
	return version, nil
}

func probeProtocolHelloWithWake(port serial.Port, timeout time.Duration) (int, error) {
	_ = port.ResetInputBuffer()
	_ = sendProbeHelloRequest(port)

	initial := minDuration(timeout, probeInitialWindow)
	if initial > 0 {
		if version, err := probeProtocolVersion(port, initial); err == nil {
			return version, nil
		}
		timeout -= initial
	}
	if timeout <= 0 {
		return 0, fmt.Errorf("no MAMA protocol hello detected")
	}

	// Some boards emit protocol hello only after reset; DTR pulse improves probe reliability.
	if err := pulseProbeDTR(port); err == nil {
		wait := minDuration(timeout, probeResetWait)
		if wait > 0 {
			time.Sleep(wait)
			timeout -= wait
		}
	}

	if timeout <= 0 {
		timeout = probeReadTimeout
	}
	_ = port.ResetInputBuffer()
	_ = sendProbeHelloRequest(port)
	return probeProtocolVersion(port, timeout)
}

func sendProbeHelloRequest(port serial.Port) error {
	if _, err := port.Write([]byte("MAMA:HELLO?\n")); err != nil {
		return err
	}
	if _, err := port.Write([]byte("MAMA:WHO?\n")); err != nil {
		return err
	}
	return nil
}

func pulseProbeDTR(port serial.Port) error {
	if err := port.SetDTR(false); err != nil {
		return err
	}
	time.Sleep(probeDTRLowDuration)
	if err := port.SetDTR(true); err != nil {
		return err
	}
	return nil
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

func probeProtocolVersion(port lineReaderPort, timeout time.Duration) (int, error) {
	deadline := time.Now().Add(timeout)
	chunk := make([]byte, readChunkSize)
	var pending bytes.Buffer

	for time.Now().Before(deadline) {
		n, err := port.Read(chunk)
		if err != nil {
			if isIdleReadError(err) {
				time.Sleep(idleReadDelay)
				continue
			}
			if errors.Is(err, io.EOF) {
				break
			}
			return 0, err
		}
		if n == 0 {
			time.Sleep(idleReadDelay)
			continue
		}

		data := chunk[:n]
		for len(data) > 0 {
			i := bytes.IndexAny(data, "\n\r\x00")
			if i < 0 {
				if _, werr := pending.Write(data); werr != nil {
					return 0, werr
				}
				if pending.Len() > maxLineBytes {
					return 0, fmt.Errorf("serial line too long (> %d bytes)", maxLineBytes)
				}
				break
			}

			if _, werr := pending.Write(data[:i]); werr != nil {
				return 0, werr
			}
			if pending.Len() > maxLineBytes {
				return 0, fmt.Errorf("serial line too long (> %d bytes)", maxLineBytes)
			}

			line := pending.String()
			pending.Reset()

			if version, ok, err := parseMAMAProtocolVersion(line, mamaHelloLinePattern); ok || err != nil {
				if err != nil {
					return 0, err
				}
				return version, nil
			}
			if version, ok, err := parseMAMAProtocolVersion(line, mamaHelloTokenPattern); ok || err != nil {
				if err != nil {
					return 0, err
				}
				return version, nil
			}

			delimiter := data[i]
			data = data[i+1:]
			if len(data) > 0 && ((delimiter == '\r' && data[0] == '\n') || (delimiter == '\n' && data[0] == '\r')) {
				data = data[1:]
			}
		}
	}

	return 0, fmt.Errorf("no MAMA protocol hello detected")
}

func parseMAMAProtocolVersion(line string, pattern *regexp.Regexp) (int, bool, error) {
	line = strings.TrimSpace(strings.ReplaceAll(line, "\x00", ""))
	matches := pattern.FindStringSubmatch(line)
	if len(matches) != 2 {
		return 0, false, nil
	}
	version, err := strconv.Atoi(strings.TrimSpace(matches[1]))
	if err != nil {
		return 0, true, fmt.Errorf("bad MAMA protocol version: %q", line)
	}
	if version <= 0 {
		return 0, true, fmt.Errorf("bad MAMA protocol version: %q", line)
	}
	return version, true, nil
}

func (r *Reader) Close() error { return r.port.Close() }

func (r *Reader) ReadLines(ctx context.Context, out chan<- string) error {
	defer close(out)
	chunk := make([]byte, readChunkSize)
	var pending bytes.Buffer

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		n, err := r.port.Read(chunk)
		if err != nil {
			if isIdleReadError(err) {
				r.idlePause()
				continue
			}
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("serial closed")
			}
			return err
		}
		if n == 0 {
			r.idlePause()
			continue
		}
		data := chunk[:n]
		for len(data) > 0 {
			i := bytes.IndexAny(data, "\n\r\x00")
			if i < 0 {
				if _, werr := pending.Write(data); werr != nil {
					return werr
				}
				if pending.Len() > maxLineBytes {
					return fmt.Errorf("serial line too long (> %d bytes)", maxLineBytes)
				}
				break
			}

			if _, werr := pending.Write(data[:i]); werr != nil {
				return werr
			}
			if pending.Len() > maxLineBytes {
				return fmt.Errorf("serial line too long (> %d bytes)", maxLineBytes)
			}

			line := pending.String()
			select {
			case out <- line:
			case <-ctx.Done():
				return ctx.Err()
			}
			pending.Reset()

			delimiter := data[i]
			data = data[i+1:]
			// Normalize CRLF / LFCR pairs as a single separator.
			if len(data) > 0 && ((delimiter == '\r' && data[0] == '\n') || (delimiter == '\n' && data[0] == '\r')) {
				data = data[1:]
			}
		}
	}
}

func (r *Reader) idlePause() {
	if r.sleep != nil {
		r.sleep(idleReadDelay)
		return
	}
	time.Sleep(idleReadDelay)
}

func isIdleReadError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrNoProgress) {
		return true
	}

	var timeoutErr interface{ Timeout() bool }
	if errors.As(err, &timeoutErr) && timeoutErr.Timeout() {
		return true
	}

	var portErr *serial.PortError
	if errors.As(err, &portErr) {
		if portErr.Code() == serial.PortClosed {
			return false
		}
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	var temporaryErr interface{ Temporary() bool }
	if errors.As(err, &temporaryErr) && temporaryErr.Temporary() {
		return true
	}

	return false
}
