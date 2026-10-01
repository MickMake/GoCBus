package runtime

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
