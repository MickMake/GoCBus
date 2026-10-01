//go:build hardware

package hardware_test

import (
	"context"
	"errors"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/MickMake/GoCBus/internal/capture"
	"github.com/MickMake/GoCBus/internal/transport"
)

func TestTransportCapture(t *testing.T) {
	if os.Getenv("GOCBUS_HARDWARE") != "1" {
		t.Skip("set GOCBUS_HARDWARE=1 for the explicit real-hardware test")
	}

	serialDevice := strings.TrimSpace(os.Getenv("GOCBUS_SERIAL"))
	tcpAddress := strings.TrimSpace(os.Getenv("GOCBUS_TCP"))
	if (serialDevice == "") == (tcpAddress == "") {
		t.Fatal("set exactly one of GOCBUS_SERIAL or GOCBUS_TCP")
	}
	capturePath := strings.TrimSpace(os.Getenv("GOCBUS_CAPTURE"))
	if capturePath == "" {
		t.Fatal("set GOCBUS_CAPTURE to an explicit local output path")
	}

	duration := 15 * time.Second
	if value := strings.TrimSpace(os.Getenv("GOCBUS_DURATION")); value != "" {
		parsed, err := time.ParseDuration(value)
		if err != nil || parsed <= 0 {
			t.Fatalf("invalid GOCBUS_DURATION %q", value)
		}
		duration = parsed
	}

	var dialer transport.Dialer
	if serialDevice != "" {
		dialer = transport.NewSerialDialer(serialDevice)
	} else {
		dialer = transport.NewTCPDialer(tcpAddress)
	}
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()
	connection, err := dialer.Dial(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()

	file, err := os.OpenFile(capturePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	writer := capture.NewWriter(file)
	var received atomic.Int64
	connection = transport.Observe(connection, func(direction capture.Direction, data []byte) error {
		if direction == capture.RX {
			received.Add(int64(len(data)))
		}
		return writer.Write(capture.Record{Timestamp: time.Now().UTC(), Direction: direction, Data: data})
	})

	go func() {
		<-ctx.Done()
		_ = connection.Close()
	}()
	buffer := make([]byte, 4096)
	for {
		_, err := connection.Read(buffer)
		if err == nil {
			continue
		}
		if ctx.Err() != nil {
			break
		}
		if errors.Is(err, io.EOF) {
			t.Fatalf("hardware transport disconnected before %s", duration)
		}
		t.Fatal(err)
	}
	if received.Load() == 0 {
		t.Fatalf("captured no RX bytes during %s", duration)
	}
	t.Logf("captured %d RX bytes in %s at %s", received.Load(), duration, capturePath)
}
