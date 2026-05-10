package bot

import (
	"context"
	"log"
	"regexp"
	"runtime/debug"
	"time"

	"discord-bot/postscraper"
	"github.com/bwmarrin/discordgo"
)

var urlPattern = regexp.MustCompile(`https?://\S+`)

// Handler wires Discord message events to the oEmbed pipeline.
type Handler struct {
	registry *postscraper.Registry
	session  *discordgo.Session
}

func New(s *discordgo.Session, r *postscraper.Registry) *Handler {
	return &Handler{session: s, registry: r}
}

// Register attaches the MessageCreate listener. Call before session.Open.
func (h *Handler) Register() {
	h.session.AddHandler(h.onMessageCreate)
}

func (h *Handler) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot {
		return
	}

	urls := urlPattern.FindAllString(m.Content, -1)
	if len(urls) == 0 {
		return
	}

	for _, rawURL := range urls {
		provider, ok := h.registry.Resolve(rawURL)
		if !ok {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		resp, err := provider.Fetch(ctx, rawURL)
		cancel()

		if err != nil {
			log.Printf("[%s] fetch %q: %v\n%s", provider.Name(), rawURL, err, debug.Stack())
			continue
		}

		embed := BuildEmbed(resp, rawURL)
		_, err = s.ChannelMessageSendComplex(m.ChannelID, &discordgo.MessageSend{
			Embeds:          []*discordgo.MessageEmbed{embed},
			Reference:       m.Reference(),
			AllowedMentions: &discordgo.MessageAllowedMentions{RepliedUser: false},
			Flags:           discordgo.MessageFlagsSuppressNotifications,
		})
		if err != nil {
			log.Printf("[%s] send embed: %v", provider.Name(), err)
		}
	}
}
