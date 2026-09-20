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
go test ./postscraper/...
```

## Architecture

The bot detects supported post URLs in Discord messages and replies with a rich embed
preview — but only when Discord has not already produced a preview of its own.
The pipeline:

```
Discord message
  → bot.Handler (extracts URLs, dispatches)
    → postscraper.Registry (resolves URL to a Provider)
      → Provider.Fetch (scrapes OG meta tags from the post page)
        → wait out PREVIEW_GRACE_PERIOD, re-read the message
          → hasOfficialEmbed? yes → stay silent
                              no  → bot.BuildEmbed → Discord channel reply
```

**`postscraper/provider.go`** — the extensibility contract. Defines the `Provider` interface (`Name`, `CanHandle`, `Fetch`), the shared `Response` struct (oEmbed 1.0-shaped fields), and `Registry`, which holds an ordered provider list and resolves URLs via first-match.

**`postscraper/threads/`** and **`postscraper/instagram/`** — providers that fetch the post page with a crawler User-Agent and parse its Open Graph / Twitter meta tags. Despite the oEmbed-shaped `Response`, no oEmbed API is called and no access token is needed.

**`bot/handler.go`** — `MessageCreate` listener. `extractURLs` pulls previewable URLs from the message, the registry resolves each, and the work moves to a goroutine (`sendPreviews`) so the gateway handler never blocks. Scrapes run first so they overlap the grace period, then the source message is re-read once via `ChannelMessage` and each URL is judged independently. Errors are logged only — nothing is posted to the channel.

**`bot/discordembed.go`** — pure, I/O-free decision helpers: `hasOfficialEmbed` (does Discord's own preview already cover this URL?), `isRichEmbed`, `makeURLKey` (host+path normalisation) and `previewsSuppressed`.

**`bot/embed.go`** — pure conversion from `*postscraper.Response` to `*discordgo.MessageEmbed`. Strips HTML tags from `resp.HTML` for the description field.

## When the bot stays silent

- Discord already attached a **rich** embed for that URL — one with an image, thumbnail, video or description. A bare title-only embed means Discord's crawler was blocked, which is exactly the case this bot covers, so the bot still posts.
- The author opted out: the link is wrapped as `<https://…>`, hidden behind a `[text](url)` masked link, or the message carries the Suppress Embeds flag.
- The message was deleted, or edited to drop the URL, during the grace period.

Embed `Type` is deliberately not inspected when matching — Discord labels auto-embeds `link`, `article`, `rich`, `image`, `gifv` or `video` depending on the site, and a wrong guess yields a duplicate preview.

## Adding a New Provider

1. Create `postscraper/<name>/<name>.go` implementing `postscraper.Provider` (`Name`, `CanHandle`, `Fetch`)
2. Register in `main.go`: `registry.Register(myprovider.New())`

Nothing else changes — the handler and registry are provider-agnostic.

## Required Discord Settings

- **Message Content Intent** must be enabled in the Developer Portal under Bot → Privileged Gateway Intents, otherwise `m.Content` is always empty in guild channels.
- The bot needs **Read Message History** in each channel. Without it the post-grace-period re-read fails with 403; the bot logs this and falls back to posting unconditionally, so duplicate previews return.

## Environment Variables

| Variable | Description |
|---|---|
| `DISCORD_BOT_TOKEN` | Discord bot token |
| `PREVIEW_GRACE_PERIOD` | How long to wait for Discord's own preview before posting one. Go duration string (e.g. `3s`, `1500ms`). Defaults to `3s`; `0` replies immediately. |
