package bilibili

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"discord-bot/postscraper"
)

// Video page and short link formats handled:
//
//	https://www.bilibili.com/video/BV1xrnz6gEVD/?p=2
//	https://m.bilibili.com/video/av117478344627500
//	https://b23.tv/{code}
var (
	videoPattern = regexp.MustCompile(
		`^https?://(?:www\.|m\.)?bilibili\.com/video/(?:(BV[0-9A-Za-z]{10})|(?i:av)(\d+))(?:[/?#]|$)`,
	)
	shortPattern = regexp.MustCompile(`^https?://b23\.tv/[0-9A-Za-z]+`)
)

const (
	apiURL    = "https://api.bilibili.com/x/web-interface/view"
	crawlerUA = "facebookexternalhit/1.1 (+http://www.facebook.com/externalhit_uatext.php)"
)

// videoID identifies a video by exactly one of a BV id or an av number.
type videoID struct {
	bvid string
	aid  int64
}

// parseVideoID extracts the video id from a bilibili.com video page URL.
func parseVideoID(rawURL string) (videoID, bool) {
	m := videoPattern.FindStringSubmatch(rawURL)
	if m == nil {
		return videoID{}, false
	}
	if m[1] != "" {
		return videoID{bvid: m[1]}, true
	}
	aid, err := strconv.ParseInt(m[2], 10, 64)
	if err != nil {
		return videoID{}, false
	}
	return videoID{aid: aid}, true
}

func (v videoID) query() string {
	if v.bvid != "" {
		return url.Values{"bvid": {v.bvid}}.Encode()
	}
	return url.Values{"aid": {strconv.FormatInt(v.aid, 10)}}.Encode()
}

type apiEnvelope struct {
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Data    videoData `json:"data"`
}

type videoData struct {
	BVID  string `json:"bvid"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Pic   string `json:"pic"`
	Owner struct {
		Mid  int64  `json:"mid"`
		Name string `json:"name"`
	} `json:"owner"`
}

// parseResponse decodes the view API envelope into a Response. A non-zero
// code is reported with its message.
func parseResponse(r io.Reader) (*postscraper.Response, error) {
	var env apiEnvelope
	if err := json.NewDecoder(r).Decode(&env); err != nil {
		return nil, fmt.Errorf("bilibili: decode response: %w", err)
	}
	if env.Code != 0 {
		return nil, fmt.Errorf("bilibili: api error %d: %s", env.Code, env.Message)
	}
	d := env.Data
	if d.Title == "" {
		return nil, fmt.Errorf("bilibili: response has no title")
	}

	res := &postscraper.Response{
		Type:            "rich",
		Version:         "1.0",
		Title:           d.Title,
		AuthorName:      d.Owner.Name,
		ProviderName:    "Bilibili",
		ProviderURL:     "https://www.bilibili.com",
		ProviderIconURL: "https://www.bilibili.com/favicon.ico",
		ThumbnailURL:    strings.Replace(d.Pic, "http://", "https://", 1),
		HTML:            d.Desc,
		URL:             "https://www.bilibili.com/video/" + d.BVID,
	}
	if d.Owner.Mid != 0 {
		res.AuthorURL = "https://space.bilibili.com/" + strconv.FormatInt(d.Owner.Mid, 10)
	}
	return res, nil
}

type Provider struct {
	httpClient *http.Client
	// noRedirectClient returns 3xx responses as-is, for resolving short links.
	noRedirectClient *http.Client
}

func New() *Provider {
	return &Provider{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		noRedirectClient: &http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (p *Provider) Name() string { return "Bilibili" }

func (p *Provider) CanHandle(rawURL string) bool {
	return videoPattern.MatchString(rawURL) || shortPattern.MatchString(rawURL)
}

func (p *Provider) Fetch(ctx context.Context, rawURL string) (*postscraper.Response, error) {
	id, ok := parseVideoID(rawURL)
	if !ok {
		resolved, err := p.resolveShortLink(ctx, rawURL)
		if err != nil {
			return nil, err
		}
		id = resolved
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL+"?"+id.query(), nil)
	if err != nil {
		return nil, fmt.Errorf("bilibili: build request: %w", err)
	}
	req.Header.Set("User-Agent", crawlerUA)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bilibili: http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bilibili: unexpected status %d", resp.StatusCode)
	}
	return parseResponse(resp.Body)
}

// resolveShortLink follows a b23.tv link one hop and reads the video id from
// the Location header.
func (p *Provider) resolveShortLink(ctx context.Context, rawURL string) (videoID, error) {
	if !shortPattern.MatchString(rawURL) {
		return videoID{}, fmt.Errorf("bilibili: unsupported url %q", rawURL)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return videoID{}, fmt.Errorf("bilibili: build request: %w", err)
	}
	req.Header.Set("User-Agent", crawlerUA)

	resp, err := p.noRedirectClient.Do(req)
	if err != nil {
		return videoID{}, fmt.Errorf("bilibili: resolve short link: %w", err)
	}
	defer resp.Body.Close()

	loc := resp.Header.Get("Location")
	if loc == "" {
		return videoID{}, fmt.Errorf("bilibili: short link %q did not redirect (status %d)", rawURL, resp.StatusCode)
	}
	id, ok := parseVideoID(loc)
	if !ok {
		return videoID{}, fmt.Errorf("bilibili: short link %q redirected to a non-video url %q", rawURL, loc)
	}
	return id, nil
}
