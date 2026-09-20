package bot

import (
	"net/url"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// hostAliases folds domains that serve the same content, so a link written one
// way still matches the embed Discord built from the other.
var hostAliases = map[string]string{
	"threads.net": "threads.com",
	"instagr.am":  "instagram.com",
}

// urlKey is a URL reduced to the parts worth comparing: host and path.
// The query is dropped so tracking suffixes such as Instagram's ?igsh= compare
// equal to the bare link; path case is preserved because Threads and Instagram
// shortcodes are case-sensitive.
type urlKey struct {
	host string
	path string
}

func makeURLKey(raw string) (urlKey, bool) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.Host == "" {
		return urlKey{}, false
	}

	host := strings.ToLower(u.Hostname())
	host = strings.TrimPrefix(host, "www.")
	if canonical, ok := hostAliases[host]; ok {
		host = canonical
	}

	return urlKey{
		host: host,
		path: strings.TrimSuffix(u.EscapedPath(), "/"),
	}, true
}

// isRichEmbed reports whether a Discord auto-embed carries real content rather
// than a bare link. Discord often attaches a title-only embed when its crawler
// is blocked — that is exactly the case this bot exists to cover, so it does
// not count as an official preview.
func isRichEmbed(e *discordgo.MessageEmbed) bool {
	if e == nil {
		return false
	}
	return e.Description != "" ||
		e.Image != nil ||
		e.Thumbnail != nil ||
		e.Video != nil
}

// hasOfficialEmbed reports whether msg already carries a rich Discord-generated
// preview covering rawURL.
//
// Embed Type is deliberately not inspected: Discord labels auto-embeds link,
// article, rich, image, gifv or video depending on the site, and a wrong guess
// here produces a duplicate preview — the bug this check exists to prevent.
// Embeds with no URL are skipped instead, since they cannot be attributed.
func hasOfficialEmbed(msg *discordgo.Message, rawURL string) bool {
	if msg == nil || len(msg.Embeds) == 0 {
		return false
	}

	want, ok := makeURLKey(rawURL)
	if !ok {
		return false
	}

	sameHost := false
	for _, e := range msg.Embeds {
		if e == nil || e.URL == "" || !isRichEmbed(e) {
			continue
		}
		got, ok := makeURLKey(e.URL)
		if !ok {
			continue
		}
		if got == want {
			return true
		}
		if got.host == want.host {
			sameHost = true
		}
	}

	// Discord canonicalises some links while unfurling — an instagram.com/share/x
	// or threads.com/t/x embed comes back pointing at the resolved post — so the
	// paths differ even though the embed really is for this link. Accept a
	// host-only match, but only when rawURL is the sole link to that host in the
	// message, so one URL's embed can never be credited to a sibling URL.
	return sameHost && countURLsOnHost(msg.Content, want.host) == 1
}

// countURLsOnHost returns how many previewable links in content point at host.
func countURLsOnHost(content, host string) int {
	n := 0
	for _, raw := range extractURLs(content) {
		if k, ok := makeURLKey(raw); ok && k.host == host {
			n++
		}
	}
	return n
}

// previewsSuppressed reports whether the author turned previews off for this
// message via Discord's "Remove Embed" action.
func previewsSuppressed(msg *discordgo.Message) bool {
	return msg != nil && msg.Flags&discordgo.MessageFlagsSuppressEmbeds != 0
}
