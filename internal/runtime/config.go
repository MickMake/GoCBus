package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
)

const defaultLogLevel = "info"

// Config contains only settings consumed by the implemented slices.
type Config struct {
	LogLevel     string `json:"log_level"`
	RawLog       bool   `json:"raw_log"`
	SerialDevice string `json:"serial_device,omitempty"`
	TCPAddress   string `json:"tcp_address,omitempty"`
	CapturePath  string `json:"capture_path,omitempty"`
}

func DefaultConfig() Config {
	return Config{LogLevel: defaultLogLevel}
}

// LoadConfig loads one JSON object and rejects unknown fields and trailing data.
func LoadConfig(path string) (Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	config := DefaultConfig()
	// Decode through a pointer so a top-level null becomes nil instead of
	// silently leaving the default struct unchanged.
	decoded := &config
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}
	if decoded == nil {
		return Config{}, errors.New("decode config: expected one JSON object")
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Config{}, errors.New("decode config: expected one JSON object")
		}
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	if err := decoded.Validate(); err != nil {
		return Config{}, err
	}
	return *decoded, nil
}

func (c Config) Validate() error {
	switch strings.ToLower(c.LogLevel) {
	case "trace", "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("invalid log_level %q: want trace, debug, info, warn, or error", c.LogLevel)
	}

	serialConfigured := strings.TrimSpace(c.SerialDevice) != ""
	tcpConfigured := strings.TrimSpace(c.TCPAddress) != ""
	if serialConfigured && tcpConfigured {
		return errors.New("configure exactly one of serial_device or tcp_address")
	}
	if tcpConfigured {
		host, port, err := net.SplitHostPort(c.TCPAddress)
		if err != nil {
			return fmt.Errorf("invalid tcp_address %q: %w", c.TCPAddress, err)
		}
		portNumber, err := strconv.Atoi(port)
		if strings.TrimSpace(host) == "" || err != nil || portNumber < 1 || portNumber > 65535 {
			return fmt.Errorf("invalid tcp_address %q: require host and numeric port 1-65535", c.TCPAddress)
		}
	}
	if strings.TrimSpace(c.CapturePath) != "" && !serialConfigured && !tcpConfigured {
		return errors.New("capture_path requires serial_device or tcp_address")
	}
	return nil
}

func (c Config) transportConfigured() bool {
	return strings.TrimSpace(c.SerialDevice) != "" || strings.TrimSpace(c.TCPAddress) != ""
}
