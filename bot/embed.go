package bot

import (
	"regexp"
	"strings"

	"discord-bot/postscraper"
	"github.com/bwmarrin/discordgo"
)

const colorThreads = 0x000000

var htmlTagPattern = regexp.MustCompile(`<[^>]*>`)

// BuildEmbed converts an oEmbed response into a Discord rich embed.
// sourceURL is passed separately so the embed title links to the original post.
func BuildEmbed(resp *postscraper.Response, sourceURL string) *discordgo.MessageEmbed {
	title := resp.Title
	if title == "" {
		title = resp.ProviderName
	}

	embed := &discordgo.MessageEmbed{
		Title: title,
		URL:   sourceURL,
		Color: colorThreads,
	}

	if resp.HTML != "" {
		plain := htmlTagPattern.ReplaceAllString(resp.HTML, "")
		plain = strings.TrimSpace(plain)
		if len(plain) > 300 {
			plain = plain[:300] + "…"
		}
		embed.Description = plain
	}

	if resp.AuthorName != "" {
		embed.Author = &discordgo.MessageEmbedAuthor{
			Name: resp.AuthorName,
			URL:  resp.AuthorURL,
		}
	}

	if resp.ThumbnailURL != "" {
		embed.Thumbnail = &discordgo.MessageEmbedThumbnail{
			URL: resp.ThumbnailURL,
		}
	}

	if resp.ProviderName != "" {
		embed.Footer = &discordgo.MessageEmbedFooter{
			Text: resp.ProviderName,
		}
	}

	return embed
}
