package runtime

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    Config
		wantErr string
	}{
		{
			name:    "valid",
			content: `{"log_level":"debug","raw_log":true}`,
			want:    Config{LogLevel: "debug", RawLog: true},
		},
		{
			name:    "defaults omitted fields",
			content: `{}`,
			want:    DefaultConfig(),
		},
		{
			name:    "unknown field",
			content: `{"verbose":true}`,
			wantErr: "unknown field",
		},
		{
			name:    "null root",
			content: `null`,
			wantErr: "expected one JSON object",
		},
		{
			name:    "array root",
			content: `[]`,
			wantErr: "cannot unmarshal array",
		},
		{
			name:    "scalar root",
			content: `true`,
			wantErr: "cannot unmarshal bool",
		},
		{
			name:    "invalid level",
			content: `{"log_level":"verbose"}`,
			wantErr: "invalid log_level",
		},
		{
			name:    "trailing object",
			content: `{} {}`,
			wantErr: "expected one JSON object",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			if err := os.WriteFile(path, []byte(tt.content), 0o600); err != nil {
				t.Fatal(err)
			}

			got, err := LoadConfig(path)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("LoadConfig() error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("LoadConfig() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("LoadConfig() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestLoadConfigMissingFile(t *testing.T) {
	_, err := LoadConfig(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil || !strings.Contains(err.Error(), "open config") {
		t.Fatalf("LoadConfig() error = %v, want open config error", err)
	}
}
