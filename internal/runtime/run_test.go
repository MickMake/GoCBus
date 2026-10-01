package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/MickMake/GoCBus/internal/capture"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name       string
		args       func(t *testing.T) []string
		wantCode   int
		wantOutput string
		wantError  string
	}{
		{
			name:       "defaults",
			args:       func(*testing.T) []string { return nil },
			wantOutput: "configuration valid",
		},
		{
			name: "file with override",
			args: func(t *testing.T) []string {
				path := filepath.Join(t.TempDir(), "config.json")
				if err := os.WriteFile(path, []byte(`{"log_level":"error"}`), 0o600); err != nil {
					t.Fatal(err)
				}
				return []string{"-config", path, "-log-level", "debug", "-raw-log"}
			},
			wantOutput: "configuration valid",
			wantError:  "raw_log=true",
		},
		{
			name:      "invalid override",
			args:      func(*testing.T) []string { return []string{"-log-level", "loud"} },
			wantCode:  1,
			wantError: "invalid log_level",
		},
		{
			name:      "help",
			args:      func(*testing.T) []string { return []string{"-help"} },
			wantError: "Usage of GoCBus:",
		},
		{
			name:      "unexpected argument",
			args:      func(*testing.T) []string { return []string{"surprise"} },
			wantCode:  2,
			wantError: "unexpected arguments",
		},
		{
			name:      "mutually exclusive CLI transports",
			args:      func(*testing.T) []string { return []string{"-serial", "/dev/pci", "-tcp", "127.0.0.1:10001"} },
			wantCode:  2,
			wantError: "mutually exclusive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := Run(tt.args(t), &stdout, &stderr)
			if code != tt.wantCode {
				t.Errorf("Run() code = %d, want %d", code, tt.wantCode)
			}
			if !strings.Contains(stdout.String(), tt.wantOutput) {
				t.Errorf("stdout = %q, want containing %q", stdout.String(), tt.wantOutput)
			}
			if !strings.Contains(stderr.String(), tt.wantError) {
				t.Errorf("stderr = %q, want containing %q", stderr.String(), tt.wantError)
			}
		})
	}
}

func TestRunTCPTransportCapturesTrafficAndReportsDisconnect(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverErr := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		_, writeErr := connection.Write([]byte{0x05, 0x38, 0xff})
		closeErr := connection.Close()
		serverErr <- errors.Join(writeErr, closeErr)
	}()

	capturePath := filepath.Join(t.TempDir(), "traffic.ndjson")
	configPath := writeConfig(t, Config{
		LogLevel:    "info",
		TCPAddress:  listener.Addr().String(),
		CapturePath: capturePath,
	})
	var stdout, stderr bytes.Buffer
	code := RunContext(context.Background(), []string{"-config", configPath}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("RunContext() code = %d, want disconnect failure; stderr = %q", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "connected to TCP CNI") || !strings.Contains(stderr.String(), "transport disconnected") {
		t.Fatalf("stdout = %q; stderr = %q", stdout.String(), stderr.String())
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}

	file, err := os.Open(capturePath)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	record, err := capture.NewReader(file).Next()
	if err != nil {
		t.Fatal(err)
	}
	if record.Direction != capture.RX || !bytes.Equal(record.Data, []byte{0x05, 0x38, 0xff}) {
		t.Fatalf("capture record = %+v", record)
	}
}

func TestRunTCPTransportStopsCleanlyOnCancellation(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	accepted := make(chan struct{})
	serverErr := make(chan error, 1)
	go func() {
		connection, err := listener.Accept()
		if err != nil {
			serverErr <- err
			return
		}
		close(accepted)
		_, err = io.Copy(io.Discard, connection)
		serverErr <- errors.Join(err, connection.Close())
	}()

	configPath := writeConfig(t, Config{LogLevel: "info", TCPAddress: listener.Addr().String()})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-accepted
		cancel()
	}()

	var stdout, stderr bytes.Buffer
	if code := RunContext(ctx, []string{"-config", configPath}, &stdout, &stderr); code != 0 {
		t.Fatalf("RunContext() code = %d; stderr = %q", code, stderr.String())
	}
	if err := <-serverErr; err != nil {
		t.Fatal(err)
	}
}

func TestRunTCPTransportStopsCleanlyWhenCancelledBeforeDial(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr bytes.Buffer
	code := RunContext(ctx, []string{"-tcp", "127.0.0.1:10001"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("RunContext() code = %d; stderr = %q", code, stderr.String())
	}
}

func TestRunAppliesTransportOverridesBeforeValidation(t *testing.T) {
	tests := []struct {
		name   string
		config func(*testing.T) Config
		args   []string
	}{
		{
			name: "CLI transport satisfies capture requirement",
			config: func(t *testing.T) Config {
				return Config{LogLevel: "info", CapturePath: filepath.Join(t.TempDir(), "traffic.ndjson")}
			},
			args: []string{"-tcp", "127.0.0.1:10001"},
		},
		{
			name: "CLI serial replaces invalid TCP address",
			config: func(*testing.T) Config {
				return Config{LogLevel: "info", TCPAddress: "stale-invalid-address"}
			},
			args: []string{"-serial", "/dev/test-pci"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := filepath.Join(t.TempDir(), "config.json")
			content, err := json.Marshal(tt.config(t))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(configPath, content, 0o600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			args := append([]string{"-config", configPath}, tt.args...)
			var stdout, stderr bytes.Buffer
			if code := RunContext(ctx, args, &stdout, &stderr); code != 0 {
				t.Fatalf("RunContext() code = %d; stderr = %q", code, stderr.String())
			}
		})
	}
}

func writeConfig(t *testing.T, config Config) string {
	t.Helper()
	data, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
