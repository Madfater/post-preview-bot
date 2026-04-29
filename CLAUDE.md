# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Install dependencies
go mod tidy

# Build
go build ./...

# Run
export $(grep -v '^#' .env | xargs)
go run .

# Run a specific test
go test ./bot/...
go test ./oembed/...
```

## Architecture

The bot detects oEmbed-supported URLs in Discord messages and replies with a rich embed preview. Three layers compose the pipeline:

```
Discord message
  → bot.Handler (extracts URLs, dispatches)
    → oembed.Registry (resolves URL to a Provider)
      → Provider.Fetch (calls upstream oEmbed API)
        → bot.BuildEmbed (converts Response → discordgo.MessageEmbed)
          → Discord channel reply
```

**`oembed/provider.go`** — the extensibility contract. Defines `Provider` interface (`Name`, `CanHandle`, `Fetch`), the shared `Response` struct (standard oEmbed 1.0 fields), and `Registry` which holds an ordered provider list and resolves URLs via first-match.

**`oembed/threads/`** — Threads oEmbed provider. Hits `https://graph.threads.net/oembed` with an `access_token`. Matches both `threads.com/@user/post/id` and `threads.com/t/id` URL formats. Returns `ErrRateLimit` on HTTP 429.

**`bot/handler.go`** — `MessageCreate` listener. Extracts all `https?://` URLs from message content, resolves each through the registry, fetches oEmbed data with an 8-second timeout, and sends embeds. Errors are logged silently (no error message sent to channel).

**`bot/embed.go`** — pure conversion from `*oembed.Response` to `*discordgo.MessageEmbed`. Strips HTML tags from `resp.HTML` for the description field.

## Adding a New oEmbed Provider

1. Create `oembed/<name>/<name>.go` implementing `oembed.Provider` (`Name`, `CanHandle`, `Fetch`)
2. Register in `main.go`: `registry.Register(myprovider.New(cfg.SomeToken))`

Nothing else changes — the handler and registry are provider-agnostic.

## Required Discord Developer Portal Settings

- **Message Content Intent** must be enabled under Bot → Privileged Gateway Intents, otherwise `m.Content` is always empty in guild channels.

## Environment Variables

| Variable | Description |
|---|---|
| `DISCORD_BOT_TOKEN` | Discord bot token |
| `META_ACCESS_TOKEN` | Meta OAuth 2.0 app access token (Threads oEmbed API, 1000 req/hour limit) |
