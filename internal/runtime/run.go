package runtime

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/MickMake/GoCBus/internal/capture"
	"github.com/MickMake/GoCBus/internal/transport"
)

type optionalString struct {
	set   bool
	value string
}

func (value *optionalString) String() string { return value.value }

func (value *optionalString) Set(input string) error {
	value.set = true
	value.value = input
	return nil
}

type optionalBool struct {
	set   bool
	value bool
}

func (value *optionalBool) String() string { return fmt.Sprint(value.value) }

func (value *optionalBool) Set(input string) error {
	switch input {
	case "true":
		value.value = true
	case "false":
		value.value = false
	default:
		return fmt.Errorf("want true or false")
	}
	value.set = true
	return nil
}

func (value *optionalBool) IsBoolFlag() bool { return true }

// Run loads the configuration and runs the raw transport until it disconnects
// or the process context is cancelled.
func Run(args []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), args, stdout, stderr)
}

func RunContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("GoCBus", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var configPath string
	var logLevel optionalString
	var rawLog optionalBool
	var serialDevice optionalString
	var tcpAddress optionalString
	var capturePath optionalString
	flags.StringVar(&configPath, "config", "", "path to a JSON configuration file")
	flags.Var(&logLevel, "log-level", "override log level: trace, debug, info, warn, or error")
	flags.Var(&rawLog, "raw-log", "override raw RX/TX trace logging (true or false)")
	flags.Var(&serialDevice, "serial", "serial PCI device path (9600 8N1)")
	flags.Var(&tcpAddress, "tcp", "TCP CNI address in host:port form")
	flags.Var(&capturePath, "capture", "write raw RX/TX traffic to this NDJSON file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "GoCBus: unexpected arguments: %v\n", flags.Args())
		return 2
	}

	config := DefaultConfig()
	if configPath != "" {
		loaded, err := decodeConfig(configPath)
		if err != nil {
			fmt.Fprintf(stderr, "GoCBus: %v\n", err)
			return 1
		}
		config = loaded
	}
	if logLevel.set {
		config.LogLevel = logLevel.value
	}
	if rawLog.set {
		config.RawLog = rawLog.value
	}
	if serialDevice.set && tcpAddress.set {
		fmt.Fprintln(stderr, "GoCBus: -serial and -tcp are mutually exclusive")
		return 2
	}
	if serialDevice.set {
		config.SerialDevice = serialDevice.value
		config.TCPAddress = ""
	}
	if tcpAddress.set {
		config.TCPAddress = tcpAddress.value
		config.SerialDevice = ""
	}
	if capturePath.set {
		config.CapturePath = capturePath.value
	}
	if err := config.Validate(); err != nil {
		fmt.Fprintf(stderr, "GoCBus: %v\n", err)
		return 1
	}

	logger, err := newLogger(stderr, config)
	if err != nil {
		fmt.Fprintf(stderr, "GoCBus: configure logging: %v\n", err)
		return 1
	}
	logger.Debug("configuration loaded", "raw_log", config.RawLog)
	if !config.transportConfigured() {
		fmt.Fprintln(stdout, "GoCBus configuration valid; transport is not configured")
		return 0
	}

	var captureFile *os.File
	var captureWriter *capture.Writer
	if strings.TrimSpace(config.CapturePath) != "" {
		file, err := capture.OpenFile(config.CapturePath)
		if err != nil {
			fmt.Fprintf(stderr, "GoCBus: open capture: %v\n", err)
			return 1
		}
		captureFile = file
		defer captureFile.Close()
		captureWriter = capture.NewWriter(captureFile)
	}

	dialer, description := configuredDialer(config)
	connection, err := dialer.Dial(ctx)
	if err != nil {
		if ctx.Err() != nil {
			logger.Info("C-Bus transport stopped", "reason", ctx.Err())
			return 0
		}
		fmt.Fprintf(stderr, "GoCBus: %v\n", err)
		return 1
	}
	defer connection.Close()

	observers := []transport.Observer{
		func(direction capture.Direction, data []byte) error {
			logger.LogRaw(string(direction), data)
			return nil
		},
	}
	if captureWriter != nil {
		observers = append(observers, func(direction capture.Direction, data []byte) error {
			return captureWriter.Write(capture.Record{
				Timestamp: time.Now().UTC(),
				Direction: direction,
				Data:      data,
			})
		})
	}
	connection = transport.Observe(connection, observers...)

	stopClose := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			_ = connection.Close()
		case <-stopClose:
		}
	}()
	defer close(stopClose)

	logger.Info("C-Bus transport connected", "transport", description)
	fmt.Fprintf(stdout, "GoCBus connected to %s\n", description)
	buffer := make([]byte, 4096)
	for {
		_, err := connection.Read(buffer)
		if err == nil {
			continue
		}
		if ctx.Err() != nil {
			logger.Info("C-Bus transport stopped", "reason", ctx.Err())
			return 0
		}
		fmt.Fprintf(stderr, "GoCBus: C-Bus transport disconnected: %v\n", err)
		return 1
	}
}

func configuredDialer(config Config) (transport.Dialer, string) {
	if strings.TrimSpace(config.SerialDevice) != "" {
		return transport.NewSerialDialer(config.SerialDevice), "serial PCI " + config.SerialDevice
	}
	return transport.NewTCPDialer(config.TCPAddress), "TCP CNI " + config.TCPAddress
}
