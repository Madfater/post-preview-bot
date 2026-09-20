package config

import (
	"testing"
	"time"
)

func TestLoadPreviewGracePeriod(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    time.Duration
		wantErr bool
	}{
		{"unset falls back to default", "", defaultPreviewGracePeriod, false},
		{"seconds", "5s", 5 * time.Second, false},
		{"milliseconds", "1500ms", 1500 * time.Millisecond, false},
		{"zero is allowed", "0s", 0, false},
		{"negative rejected", "-1s", 0, true},
		{"unparseable rejected", "soon", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DISCORD_BOT_TOKEN", "test-token")
			t.Setenv("PREVIEW_GRACE_PERIOD", tt.raw)

			cfg, err := Load()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("Load() with PREVIEW_GRACE_PERIOD=%q: want error, got none", tt.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("Load() with PREVIEW_GRACE_PERIOD=%q: %v", tt.raw, err)
			}
			if cfg.PreviewGracePeriod != tt.want {
				t.Errorf("PreviewGracePeriod = %v, want %v", cfg.PreviewGracePeriod, tt.want)
			}
		})
	}
}

func TestLoadRequiresToken(t *testing.T) {
	t.Setenv("DISCORD_BOT_TOKEN", "")
	if _, err := Load(); err == nil {
		t.Error("Load() without DISCORD_BOT_TOKEN: want error, got none")
	}
}
