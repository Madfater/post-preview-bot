package threads

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

// matches both Threads URL formats:
//
//	https://www.threads.com/@username/post/shortcode/
//	https://www.threads.com/t/shortcode/
//	and threads.net equivalents
var urlPattern = regexp.MustCompile(
	`https?://(?:www\.)?threads\.(?:com|net)/(?:@[\w.]+/post/[\w-]+|t/[\w-]+)/?`,
)

var usernamePattern = regexp.MustCompile(`/@([\w.]+)/post/`)

type Provider struct {
	httpClient *http.Client
}

func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (p *Provider) Name() string { return "Threads" }

func (p *Provider) CanHandle(rawURL string) bool {
	return urlPattern.MatchString(rawURL)
}

func (p *Provider) Fetch(ctx context.Context, rawURL string) (*postscraper.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("threads: build request: %w", err)
	}
	// Use Meta's own crawler UA so Threads serves pre-rendered OG meta tags.
	req.Header.Set("User-Agent", "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("threads: http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("threads: unexpected status %d", resp.StatusCode)
	}

	og, err := parseOGTags(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("threads: parse HTML: %w", err)
	}

	title := og["og:title"]
	if title == "" {
		return nil, fmt.Errorf("threads: no og:title found in page")
	}

	authorName, authorURL := extractAuthor(rawURL)

	return &postscraper.Response{
		Type:         "rich",
		Version:      "1.0",
		Title:        title,
		AuthorName:   authorName,
		AuthorURL:    authorURL,
		ProviderName: "Threads",
		ProviderURL:  "https://www.threads.com",
		ThumbnailURL: og["og:image"],
		HTML:         og["og:description"],
	}, nil
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
			if strings.HasPrefix(prop, "og:") && content != "" {
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

func extractAuthor(rawURL string) (name, profileURL string) {
	m := usernamePattern.FindStringSubmatch(rawURL)
	if m == nil {
		return "", ""
	}
	username := m[1]
	return "@" + username, "https://www.threads.com/@" + username
}
