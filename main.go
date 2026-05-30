package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"discord-bot/bot"
	"discord-bot/config"
	"discord-bot/postscraper"
	"discord-bot/postscraper/instagram"
	"discord-bot/postscraper/threads"
	"github.com/bwmarrin/discordgo"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	registry := postscraper.NewRegistry()

	if err := registry.Register(threads.New()); err != nil {
		log.Fatalf("register threads provider: %v", err)
	}
	if err := registry.Register(instagram.New()); err != nil {
		log.Fatalf("register instagram provider: %v", err)
	}

	dg, err := discordgo.New("Bot " + cfg.DiscordToken)
	if err != nil {
		log.Fatalf("discord session: %v", err)
	}

	// MESSAGE_CONTENT is privileged — enable it in the Discord Developer Portal
	// under Bot → Privileged Gateway Intents.
	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentDirectMessages |
		discordgo.IntentMessageContent

	bot.New(dg, registry).Register()

	if err := dg.Open(); err != nil {
		log.Fatalf("discord open: %v", err)
	}
	defer dg.Close()

	log.Println("Bot running. Press Ctrl+C to stop.")
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("Shutting down.")
}
