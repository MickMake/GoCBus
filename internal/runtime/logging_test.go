package runtime

import (
	"bytes"
	"strings"
	"testing"
)

func TestLogRaw(t *testing.T) {
	var output bytes.Buffer
	logger, err := newLogger(&output, Config{LogLevel: "info", RawLog: true})
	if err != nil {
		t.Fatal(err)
	}

	logger.LogRaw("rx", []byte{0x05, 0x38, 0xff})
	got := output.String()
	for _, want := range []string{"level=DEBUG-4", `msg="raw traffic"`, "direction=RX", "data=0538FF"} {
		if !strings.Contains(got, want) {
			t.Errorf("log output %q does not contain %q", got, want)
		}
	}
}

func TestLogRawDisabled(t *testing.T) {
	var output bytes.Buffer
	logger, err := newLogger(&output, DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	logger.LogRaw("tx", []byte{1})
	if output.Len() != 0 {
		t.Fatalf("disabled raw logging wrote %q", output.String())
	}
}

func TestRawLogDoesNotLowerOrdinaryLogLevel(t *testing.T) {
	var output bytes.Buffer
	logger, err := newLogger(&output, Config{LogLevel: "error", RawLog: true})
	if err != nil {
		t.Fatal(err)
	}

	logger.Debug("ordinary debug")
	logger.Info("ordinary info")
	logger.LogRaw("tx", []byte{0xa3, 0x21})

	got := output.String()
	if !strings.Contains(got, `msg="raw traffic"`) || !strings.Contains(got, "direction=TX") || !strings.Contains(got, "data=A321") {
		t.Errorf("raw trace missing from output %q", got)
	}
	for _, unwanted := range []string{"ordinary debug", "ordinary info"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("output %q contains filtered message %q", got, unwanted)
		}
	}
}
