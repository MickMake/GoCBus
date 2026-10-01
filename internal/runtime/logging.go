package runtime

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// LevelTrace keeps raw traffic below ordinary debug diagnostics.
const LevelTrace = slog.Level(-8)

// Logger keeps ordinary diagnostic filtering independent from optional raw
// traffic trace output.
type Logger struct {
	*slog.Logger
	raw *slog.Logger
}

func parseLogLevel(value string) (slog.Level, error) {
	switch strings.ToLower(value) {
	case "trace":
		return LevelTrace, nil
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("invalid log level %q", value)
	}
}

func newLogger(output io.Writer, config Config) (*Logger, error) {
	level, err := parseLogLevel(config.LogLevel)
	if err != nil {
		return nil, err
	}
	result := &Logger{
		Logger: slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: level})),
	}
	if config.RawLog {
		result.raw = slog.New(slog.NewTextHandler(output, &slog.HandlerOptions{Level: LevelTrace}))
	}
	return result, nil
}

// LogRaw records protocol-agnostic traffic in a consistent, searchable form.
// Raw logging is separately gated because captures may contain sensitive data.
func (logger *Logger) LogRaw(direction string, data []byte) {
	if logger.raw == nil {
		return
	}
	logger.raw.Log(context.Background(), LevelTrace, "raw traffic", "direction", strings.ToUpper(direction), "data", fmt.Sprintf("%X", data))
}
