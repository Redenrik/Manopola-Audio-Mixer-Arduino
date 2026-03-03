package serialx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"go.bug.st/serial"
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
	maxLineBytes  = 1024 * 1024
	readChunkSize = 256
	idleReadDelay = 5 * time.Millisecond
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
