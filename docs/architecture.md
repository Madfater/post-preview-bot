# Architecture

Navigation map of the codebase. When this file and the code disagree, the code wins, so fix this file.

## Overview

A Discord bot that watches messages for Threads, Instagram and Bilibili post links and replies with a rich embed preview. It scrapes the post page's Open Graph tags itself (Bilibili: its public video-info JSON API), waits a grace period, and posts only for links Discord did not already preview. Providers sit behind one interface, so the handler does not know which sites exist.

## Layout

| Path | Responsibility |
| --- | --- |
| `main.go` | Loads config, registers providers, opens the Discord session with the message content intent |
| `config/` | Reads `DISCORD_BOT_TOKEN` and `PREVIEW_GRACE_PERIOD` from the environment or `.env` |
| `postscraper/provider.go` | `Provider` interface, oEmbed-shaped `Response`, first-match `Registry` |
| `postscraper/threads/` | Threads provider: fetches the post page with a crawler User-Agent, parses OG tags |
| `postscraper/instagram/` | Instagram provider: same approach, rewrites `/reels/` to `/reel/` before fetching |
| `postscraper/bilibili/` | Bilibili provider: reads the public video-info JSON API, resolves `b23.tv` short links with one un-followed redirect |
| `bot/handler.go` | `MessageCreate` listener: URL extraction, scraping, grace period, re-read, reply |
| `bot/discordembed.go` | Pure decision helpers: `hasOfficialEmbed`, `isRichEmbed`, `makeURLKey`, `previewsSuppressed` |
| `bot/embed.go` | Pure conversion from `*postscraper.Response` to `*discordgo.MessageEmbed` |
| `Dockerfile`, `docker-compose.yml` | Container build and local run |
| `.github/workflows/docker-publish.yml` | Go build and test on PRs; on `main`, image push to GHCR then a redeploy webhook |
| `.github/workflows/harness.yml` | Harness conventions and checks on PRs |
| `script/` | Harness scripts: checks, GitHub client, apply |
| `docs/` | Knowledge |

## Data flow

1. `onMessageCreate` drops bot authors and messages with the Suppress Embeds flag.
2. `extractURLs` pulls URLs from the content, skipping `<https://...>` wrapped and `[text](url)` masked links. `Registry.Resolve` keeps the ones a provider can handle.
3. The rest runs in a goroutine (`sendPreviews`). Every candidate is fetched first, so the scraping overlaps the grace period.
4. After `PREVIEW_GRACE_PERIOD`, `verify` re-reads the message once with `ChannelMessage`. A 404 means the message was deleted, and a Suppress Embeds flag set during the wait means the author opted out; both keep the bot silent. Any other error falls back to the gateway copy and posts.
5. For each URL still in the content with no rich official embed, `BuildEmbed` builds the reply. It is sent as a reply with notifications suppressed and no mention ping.

## Invariants

- The gateway handler never blocks: sleeping and network work happen in the `sendPreviews` goroutine.
- Errors are only logged; the bot never posts an error message to a channel.
- Embed `Type` is never inspected when matching official embeds; only richness (image, thumbnail, video or description) counts. A title-only embed means Discord's crawler was blocked, so the bot still posts.
- URLs are matched to embeds by host and path via `makeURLKey`: the query, a trailing slash and `www.` are dropped, and `threads.net`, `instagr.am`, `m.bilibili.com` and `b23.tv` fold into their canonical hosts. Path case is kept because shortcodes are case-sensitive. When paths differ because Discord canonicalised a share or short link, a host-only match counts, but only if that URL is the sole link to its host in the message.
- A failed re-read other than 404 degrades to posting, not to silence.
- Adding a provider only touches its own package and one `registry.Register` line in `main.go`.
