package instagram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"discord-bot/postscraper"

	"golang.org/x/net/html"
)

// matches common Instagram post URL formats:
//
//	https://www.instagram.com/p/{shortcode}/
//	https://www.instagram.com/reel/{shortcode}/
//	https://www.instagram.com/reels/{shortcode}/
//	https://www.instagram.com/tv/{shortcode}/
//	https://www.instagram.com/{username}/p|reel|reels|tv/{shortcode}/
//	https://www.instagram.com/share/{code}
var urlPattern = regexp.MustCompile(
	`https?://(?:www\.)?instagram\.com/(?:[\w.]+/)?(?:p|reel|reels|tv|share)/[\w-]+/?`,
)

var usernamePattern = regexp.MustCompile(`instagram\.com/([\w.]+)/(?:p|reel|reels|tv)/`)

// Instagram's og:description always wraps the caption with
// `N likes, M comments - user on date: "..."`. captionPattern peels
// the prefix and the surrounding quotes off so the embed shows the
// caption body only.
var captionPattern = regexp.MustCompile(`(?s)^.*?: "(.*)"\.?\s*$`)

type Provider struct {
	httpClient *http.Client
}

func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Provider) Name() string { return "Instagram" }

func (p *Provider) CanHandle(rawURL string) bool {
	return urlPattern.MatchString(rawURL)
}

func (p *Provider) Fetch(ctx context.Context, rawURL string) (*postscraper.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("instagram: build request: %w", err)
	}
	// Use Meta's own crawler UA so Instagram serves pre-rendered OG meta tags
	// instead of the JS-rendered login wall.
	req.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("instagram: http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("instagram: unexpected status %d", resp.StatusCode)
	}

	og, err := parseOGTags(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("instagram: parse HTML: %w", err)
	}

	title := og["twitter:title"]
	if title == "" {
		return nil, fmt.Errorf("instagram: no og:title found in page")
	}

	authorName, authorURL := extractAuthor(rawURL, title)

	return &postscraper.Response{
		Type:            "rich",
		Version:         "1.0",
		Title:           title,
		AuthorName:      authorName,
		AuthorURL:       authorURL,
		ProviderName:    "Instagram",
		ProviderURL:     "https://www.instagram.com",
		ProviderIconURL: "https://static.cdninstagram.com/rsrc.php/v3/yt/r/30PrGfR3xhB.png",
		ThumbnailURL:    og["og:image"],
		HTML:            cleanCaption(og["og:description"]),
	}, nil
}

func cleanCaption(s string) string {
	if m := captionPattern.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return s
}

func parseOGTags(r io.Reader) (map[string]string, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	og := map[string]string{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "meta" {
			var prop, content string
			for _, a := range n.Attr {
				switch a.Key {
				case "property", "name":
					prop = a.Val
				case "content":
					content = a.Val
				}
			}
			if strings.HasPrefix(prop, "og:") || strings.HasPrefix(prop, "twitter:") && content != "" {
				og[prop] = content
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return og, nil
}

// extractAuthor pulls the username from the URL when present
// (e.g. /username/p/code/). For canonical /p/, /reel/, /tv/, /share/
// paths the username is not in the URL, so fall back to parsing the
// og:title prefix "Username (@handle) on Instagram: ...".
func extractAuthor(rawURL, ogTitle string) (name, profileURL string) {
	if m := usernamePattern.FindStringSubmatch(rawURL); m != nil {
		username := m[1]
		return "@" + username, "https://www.instagram.com/" + username
	}
	if i := strings.Index(ogTitle, "(@"); i != -1 {
		rest := ogTitle[i+2:]
		if j := strings.Index(rest, ")"); j != -1 {
			username := rest[:j]
			if username != "" {
				return "@" + username, "https://www.instagram.com/" + username
			}
		}
	}
	return "", ""
}
