package serialx

import (
	"bufio"
	"context"
	"fmt"
	"time"

	"go.bug.st/serial"
)

type Reader struct {
	port serial.Port
	sc   *bufio.Scanner
}

func ListPorts() ([]string, error) {
	return serial.GetPortsList()
}

func Open(portName string, baud int) (*Reader, error) {
	mode := &serial.Mode{BaudRate: baud}
	p, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}
	// Some boards reset on open; short grace helps.
	time.Sleep(1200 * time.Millisecond)

	sc := bufio.NewScanner(p)
	sc.Buffer(make([]byte, 0, 1024), 1024*1024)

	return &Reader{port: p, sc: sc}, nil
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

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if !r.sc.Scan() {
			if err := r.sc.Err(); err != nil {
				return err
			}
			return fmt.Errorf("serial closed")
		}
		out <- r.sc.Text()
	}
}
