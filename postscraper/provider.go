package postscraper

import (
	"context"
)

// Response holds standard oEmbed 1.0 fields shared by all providers.
type Response struct {
	Type         string `json:"type"`
	Version      string `json:"version"`
	Title        string `json:"title,omitempty"`
	AuthorName   string `json:"author_name,omitempty"`
	AuthorURL    string `json:"author_url,omitempty"`
	ProviderName    string `json:"provider_name,omitempty"`
	ProviderURL     string `json:"provider_url,omitempty"`
	ProviderIconURL string `json:"provider_icon_url,omitempty"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	ThumbnailW   int    `json:"thumbnail_width,omitempty"`
	ThumbnailH   int    `json:"thumbnail_height,omitempty"`
	Width        int    `json:"width,omitempty"`
	Height       int    `json:"height,omitempty"`
	HTML         string `json:"html,omitempty"`
	URL          string `json:"url,omitempty"`
}

// Provider is the interface every oEmbed source must implement.
// Adding a new source requires only implementing this interface and
// calling Registry.Register once in main.
type Provider interface {
	Name() string
	CanHandle(rawURL string) bool
	Fetch(ctx context.Context, rawURL string) (*Response, error)
}

// Registry resolves URLs to the first registered Provider that can handle them.
type Registry struct {
	providers []Provider
}

func NewRegistry() *Registry {
	return &Registry{}
}

// Register appends a provider after verifying connectivity if it implements Pinger.
func (r *Registry) Register(p Provider) error {
	r.providers = append(r.providers, p)
	return nil
}

// Resolve returns the first provider whose CanHandle returns true, or (nil, false).
func (r *Registry) Resolve(rawURL string) (Provider, bool) {
	for _, p := range r.providers {
		if p.CanHandle(rawURL) {
			return p, true
		}
	}
	return nil, false
}
