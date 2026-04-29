package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DiscordToken string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		DiscordToken: os.Getenv("DISCORD_BOT_TOKEN"),
	}

	if cfg.DiscordToken == "" {
		return nil, fmt.Errorf("missing required environment variable: DISCORD_BOT_TOKEN")
	}
	return cfg, nil
}
