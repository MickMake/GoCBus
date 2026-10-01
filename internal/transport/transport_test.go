package transport

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"go.bug.st/serial"

	"github.com/MickMake/GoCBus/internal/capture"
)

type fakePort struct {
	bytes.Buffer
	closed bool
}

func (port *fakePort) SetMode(*serial.Mode) error                           { return nil }
func (port *fakePort) Drain() error                                         { return nil }
func (port *fakePort) ResetInputBuffer() error                              { return nil }
func (port *fakePort) ResetOutputBuffer() error                             { return nil }
func (port *fakePort) SetDTR(bool) error                                    { return nil }
func (port *fakePort) SetRTS(bool) error                                    { return nil }
func (port *fakePort) GetModemStatusBits() (*serial.ModemStatusBits, error) { return nil, nil }
func (port *fakePort) SetReadTimeout(time.Duration) error                   { return nil }
func (port *fakePort) Break(time.Duration) error                            { return nil }
func (port *fakePort) Close() error {
	port.closed = true
	return nil
}

func TestSerialDialerUsesPCISettings(t *testing.T) {
	port := &fakePort{}
	dialer := &SerialDialer{
		Device: "/dev/test-pci",
		open: func(device string, mode *serial.Mode) (serial.Port, error) {
			if device != "/dev/test-pci" {
				t.Errorf("device = %q", device)
			}
			want := serial.Mode{BaudRate: 9600, DataBits: 8, Parity: serial.NoParity, StopBits: serial.OneStopBit}
			if *mode != want {
				t.Errorf("mode = %+v, want %+v", *mode, want)
			}
			return port, nil
		},
	}

	connection, err := dialer.Dial(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if connection != port {
		t.Fatalf("connection = %T, want fake port", connection)
	}
}

func TestSerialDialerRejectsCancelledContextBeforeOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	dialer := &SerialDialer{
		Device: "/dev/test-pci",
		open: func(string, *serial.Mode) (serial.Port, error) {
			called = true
			return &fakePort{}, nil
		},
	}

	_, err := dialer.Dial(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Dial() error = %v, want context cancellation", err)
	}
	if called {
		t.Fatal("serial opener called for cancelled context")
	}
}

func TestSerialDialerClosesPortWhenCancelledDuringOpen(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	port := &fakePort{}
	dialer := &SerialDialer{
		Device: "/dev/test-pci",
		open: func(string, *serial.Mode) (serial.Port, error) {
			cancel()
			return port, nil
		},
	}

	_, err := dialer.Dial(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Dial() error = %v, want context cancellation", err)
	}
	if !port.closed {
		t.Fatal("port remained open after cancellation")
	}
}

func TestDialersRejectMissingOrMalformedTargets(t *testing.T) {
	tests := []struct {
		name   string
		dialer Dialer
		want   string
	}{
		{name: "empty serial device", dialer: NewSerialDialer(""), want: "device is required"},
		{name: "missing TCP port", dialer: NewTCPDialer("example.test"), want: "invalid address"},
		{name: "non-numeric TCP port", dialer: NewTCPDialer("example.test:https"), want: "numeric port"},
		{name: "empty TCP host", dialer: NewTCPDialer(":10001"), want: "require host"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.dialer.Dial(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Dial() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestTCPDialerReadWriteDisconnectAndReconnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		for range 2 {
			connection, err := listener.Accept()
			if err != nil {
				serverErr <- err
				return
			}
			data := make([]byte, 4)
			if _, err := io.ReadFull(connection, data); err != nil {
				serverErr <- err
				return
			}
			if string(data) != "ping" {
				serverErr <- errors.New("unexpected client bytes")
				return
			}
			if _, err := connection.Write([]byte("pong")); err != nil {
				serverErr <- err
				return
			}
			if err := connection.Close(); err != nil {
				serverErr <- err
				return
			}
		}
		serverErr <- nil
	}()

	dialer := NewTCPDialer(listener.Addr().String())
	for attempt := range 2 {
		connection, err := dialer.Dial(context.Background())
		if err != nil {
			t.Fatalf("attempt %d: %v", attempt, err)
		}
		if _, err := connection.Write([]byte("ping")); err != nil {
			t.Fatalf("attempt %d write: %v", attempt, err)
		}
		response := make([]byte, 4)
		if _, err := io.ReadFull(connection, response); err != nil {
			t.Fatalf("attempt %d read: %v", attempt, err)
		}
		if string(response) != "pong" {
			t.Fatalf("attempt %d response = %q", attempt, response)
		}
		one := make([]byte, 1)
		if _, err := connection.Read(one); !errors.Is(err, io.EOF) {
			t.Fatalf("attempt %d disconnect error = %v, want EOF", attempt, err)
		}
		_ = connection.Close()
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestTCPDialerHonoursCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewTCPDialer("127.0.0.1:1").Dial(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Dial() error = %v, want context cancellation", err)
	}
}

type partialConnection struct {
	readData  []byte
	writeData []byte
}

func (connection *partialConnection) Read(buffer []byte) (int, error) {
	return copy(buffer, connection.readData), io.EOF
}

func (connection *partialConnection) Write(data []byte) (int, error) {
	count := len(data) - 1
	connection.writeData = append(connection.writeData, data[:count]...)
	return count, io.ErrShortWrite
}

func (*partialConnection) Close() error { return nil }

func TestObserveRecordsCompletedPartialIO(t *testing.T) {
	base := &partialConnection{readData: []byte("reply")}
	var mu sync.Mutex
	var records []capture.Record
	connection := Observe(base, func(direction capture.Direction, data []byte) error {
		mu.Lock()
		defer mu.Unlock()
		records = append(records, capture.Record{Direction: direction, Data: append([]byte(nil), data...)})
		return nil
	})

	buffer := make([]byte, 8)
	if count, err := connection.Read(buffer); count != 5 || !errors.Is(err, io.EOF) {
		t.Fatalf("Read() = %d, %v", count, err)
	}
	if count, err := connection.Write([]byte("request")); count != 6 || !errors.Is(err, io.ErrShortWrite) {
		t.Fatalf("Write() = %d, %v", count, err)
	}

	if len(records) != 2 || records[0].Direction != capture.RX || string(records[0].Data) != "reply" || records[1].Direction != capture.TX || string(records[1].Data) != "reques" {
		t.Fatalf("records = %+v", records)
	}
}

func TestObservePropagatesObserverFailure(t *testing.T) {
	connection := Observe(&partialConnection{readData: []byte("x")}, func(capture.Direction, []byte) error {
		return errors.New("disk full")
	})
	buffer := make([]byte, 1)
	count, err := connection.Read(buffer)
	if count != 1 || err == nil || !strings.Contains(err.Error(), "disk full") || !errors.Is(err, io.EOF) {
		t.Fatalf("Read() = %d, %v", count, err)
	}
}
