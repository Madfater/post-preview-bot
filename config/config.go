package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

// defaultPreviewGracePeriod is how long to wait for Discord's own link preview
// before the bot decides to post one. Discord's crawler usually lands in 0.5-3s.
const defaultPreviewGracePeriod = 3 * time.Second

type Config struct {
	DiscordToken string
	// PreviewGracePeriod is the wait before re-checking whether Discord already
	// attached a preview. Zero means reply immediately.
	PreviewGracePeriod time.Duration
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DiscordToken:       os.Getenv("DISCORD_BOT_TOKEN"),
		PreviewGracePeriod: defaultPreviewGracePeriod,
	}

	if cfg.DiscordToken == "" {
		return nil, fmt.Errorf("missing required environment variable: DISCORD_BOT_TOKEN")
	}

	if raw := os.Getenv("PREVIEW_GRACE_PERIOD"); raw != "" {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid PREVIEW_GRACE_PERIOD %q: %w", raw, err)
		}
		if d < 0 {
			return nil, fmt.Errorf("invalid PREVIEW_GRACE_PERIOD %q: must not be negative", raw)
		}
		cfg.PreviewGracePeriod = d
	}

	return cfg, nil
}
