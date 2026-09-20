package bot

import (
	"context"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"discord-bot/postscraper"
	"github.com/bwmarrin/discordgo"
)

var urlPattern = regexp.MustCompile(`https?://\S+`)

// trailingJunk is punctuation the greedy \S+ match swallows from prose.
const trailingJunk = `.,!?;:'")]}>`

// fetchTimeout bounds a single provider scrape.
const fetchTimeout = 8 * time.Second

// Handler wires Discord message events to the post-scraping pipeline.
type Handler struct {
	registry *postscraper.Registry
	session  *discordgo.Session
	// grace is how long to wait for Discord to attach its own preview before
	// deciding the bot should post one.
	grace time.Duration
}

func New(s *discordgo.Session, r *postscraper.Registry, grace time.Duration) *Handler {
	return &Handler{session: s, registry: r, grace: grace}
}

// Register attaches the MessageCreate listener. Call before session.Open.
func (h *Handler) Register() {
	h.session.AddHandler(h.onMessageCreate)
}

// candidate pairs a URL found in a message with the provider that handles it.
type candidate struct {
	rawURL   string
	provider postscraper.Provider
}

// result carries a successful scrape through the grace period wait.
type result struct {
	candidate
	resp *postscraper.Response
}

// extractURLs returns the previewable URLs in content, in order and without
// duplicates. Links the author wrapped in angle brackets or hid behind a masked
// link are left out: Discord never previews those, so their lack of an embed is
// not a gap for this bot to fill. Trailing punctuation picked up from prose is
// trimmed so the URL compares cleanly against Discord's own embed URL.
func extractURLs(content string) []string {
	matches := urlPattern.FindAllStringIndex(content, -1)
	urls := make([]string, 0, len(matches))
	seen := make(map[string]bool, len(matches))

	for _, loc := range matches {
		raw := content[loc[0]:loc[1]]

		if strings.HasSuffix(raw, ">") && loc[0] > 0 && content[loc[0]-1] == '<' {
			continue // <https://…>
		}
		if loc[0] >= 2 && content[loc[0]-2:loc[0]] == "](" {
			continue // [text](https://…)
		}

		if raw = strings.TrimRight(raw, trailingJunk); raw == "" || seen[raw] {
			continue
		}
		seen[raw] = true
		urls = append(urls, raw)
	}
	return urls
}

func (h *Handler) onMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author == nil || m.Author.Bot {
		return
	}
	if previewsSuppressed(m.Message) {
		return
	}

	var candidates []candidate
	for _, rawURL := range extractURLs(m.Content) {
		if provider, ok := h.registry.Resolve(rawURL); ok {
			candidates = append(candidates, candidate{rawURL: rawURL, provider: provider})
		}
	}
	if len(candidates) == 0 {
		return
	}

	// Off the gateway goroutine: sendPreviews sleeps out the grace period.
	go h.sendPreviews(s, m.Message, candidates)
}

// sendPreviews scrapes every candidate, waits for Discord to finish unfurling,
// then replies only for the URLs Discord left without a preview of its own.
func (h *Handler) sendPreviews(s *discordgo.Session, src *discordgo.Message, candidates []candidate) {
	deadline := time.Now().Add(h.grace)

	// Scrape first so the network work overlaps the grace period.
	results := make([]result, 0, len(candidates))
	for _, c := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), fetchTimeout)
		resp, err := c.provider.Fetch(ctx, c.rawURL)
		cancel()

		if err != nil {
			log.Printf("[%s] fetch %q: %v", c.provider.Name(), c.rawURL, err)
			continue
		}
		results = append(results, result{candidate: c, resp: resp})
	}
	if len(results) == 0 {
		return
	}

	if remaining := time.Until(deadline); remaining > 0 {
		time.Sleep(remaining)
	}

	msg, ok := h.verify(s, src)
	if !ok {
		return
	}

	for _, r := range results {
		// The URL may have been edited out during the grace period.
		if msg.Content != "" && !strings.Contains(msg.Content, r.rawURL) {
			continue
		}
		if hasOfficialEmbed(msg, r.rawURL) {
			continue
		}

		embed := BuildEmbed(r.resp, r.rawURL)
		_, err := s.ChannelMessageSendComplex(msg.ChannelID, &discordgo.MessageSend{
			Embeds:          []*discordgo.MessageEmbed{embed},
			Reference:       src.Reference(),
			AllowedMentions: &discordgo.MessageAllowedMentions{RepliedUser: false},
			Flags:           discordgo.MessageFlagsSuppressNotifications,
		})
		if err != nil {
			log.Printf("[%s] send embed: %v", r.provider.Name(), err)
		}
	}
}

// verify re-reads the source message once the grace period has elapsed, so the
// decision is made against the embeds Discord has actually attached by now.
// A false second return means stay silent entirely.
func (h *Handler) verify(s *discordgo.Session, src *discordgo.Message) (*discordgo.Message, bool) {
	msg, err := s.ChannelMessage(src.ChannelID, src.ID)
	if err != nil {
		var restErr *discordgo.RESTError
		if errors.As(err, &restErr) && restErr.Response != nil &&
			restErr.Response.StatusCode == http.StatusNotFound {
			return nil, false // deleted during the grace period
		}
		// Most likely a missing Read Message History permission, otherwise a
		// transient failure. Fall back to the gateway copy and post: a
		// permission gap should degrade to the old behaviour, not silence the bot.
		log.Printf("verify message %s: %v — posting without verification", src.ID, err)
		return src, true
	}

	if previewsSuppressed(msg) {
		return nil, false
	}
	return msg, true
}
