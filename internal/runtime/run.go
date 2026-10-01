package runtime

import (
	"errors"
	"flag"
	"fmt"
	"io"
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

// Run loads and validates the foundation configuration. Transport and service
// orchestration intentionally arrive in later slices.
func Run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("GoCBus", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var configPath string
	var logLevel optionalString
	var rawLog optionalBool
	flags.StringVar(&configPath, "config", "", "path to a JSON configuration file")
	flags.Var(&logLevel, "log-level", "override log level: trace, debug, info, warn, or error")
	flags.Var(&rawLog, "raw-log", "override raw RX/TX trace logging (true or false)")
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
		loaded, err := LoadConfig(configPath)
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
	fmt.Fprintln(stdout, "GoCBus configuration valid; transport is not implemented in v0.42.1")
	return 0
}
