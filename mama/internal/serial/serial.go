package serialx

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"go.bug.st/serial"
)

type Reader struct {
	port serial.Port
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

	return &Reader{port: p}, nil
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
			if errors.Is(err, io.EOF) {
				return fmt.Errorf("serial closed")
			}
			return err
		}
		if n == 0 {
			time.Sleep(idleReadDelay)
			continue
		}
		data := chunk[:n]
		for len(data) > 0 {
			i := bytes.IndexByte(data, '\n')
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

			line := strings.TrimSuffix(pending.String(), "\r")
			select {
			case out <- line:
			case <-ctx.Done():
				return ctx.Err()
			}
			pending.Reset()
			data = data[i+1:]
		}
	}
}
