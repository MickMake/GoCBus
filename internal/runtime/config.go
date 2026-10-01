package runtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const defaultLogLevel = "info"

// Config contains only the process settings used by the foundation slice.
// Later slices extend it when they introduce settings with real consumers.
type Config struct {
	LogLevel string `json:"log_level"`
	RawLog   bool   `json:"raw_log"`
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
	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&config); err != nil {
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return Config{}, errors.New("decode config: expected one JSON object")
		}
		return Config{}, fmt.Errorf("decode config: %w", err)
	}

	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	switch strings.ToLower(c.LogLevel) {
	case "trace", "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("invalid log_level %q: want trace, debug, info, warn, or error", c.LogLevel)
	}
}
