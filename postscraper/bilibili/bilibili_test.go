package bilibili

import (
	"strings"
	"testing"
)

func TestCanHandle(t *testing.T) {
	p := New()
	tests := []struct {
		name string
		url  string
		want bool
	}{
		{"bv www", "https://www.bilibili.com/video/BV1xrnz6gEVD", true},
		{"bv trailing slash and query", "https://www.bilibili.com/video/BV1xrnz6gEVD/?p=2&share_source=copy_web", true},
		{"bv no www", "https://bilibili.com/video/BV1xrnz6gEVD", true},
		{"bv mobile", "https://m.bilibili.com/video/BV1xrnz6gEVD", true},
		{"av", "https://www.bilibili.com/video/av117478344627500", true},
		{"av uppercase", "https://www.bilibili.com/video/AV170001/", true},
		{"short link", "https://b23.tv/BV1xrnz6gEVD", true},
		{"short link code", "https://b23.tv/aB3dE9z?share_source=copy_web", true},
		{"bv too short", "https://www.bilibili.com/video/BV1xrnz6gEV", false},
		{"av without digits", "https://www.bilibili.com/video/av", false},
		{"space", "https://space.bilibili.com/12345", false},
		{"live", "https://live.bilibili.com/12345", false},
		{"dynamic", "https://t.bilibili.com/12345", false},
		{"bangumi", "https://www.bilibili.com/bangumi/play/ep1", false},
		{"b23 without code", "https://b23.tv/", false},
		{"other site", "https://example.com/video/BV1xrnz6gEVD", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := p.CanHandle(tt.url); got != tt.want {
				t.Errorf("CanHandle(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}

func TestParseVideoID(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantQuery string
		wantOK    bool
	}{
		{"bv", "https://www.bilibili.com/video/BV1xrnz6gEVD/?share_source=copy_web", "bvid=BV1xrnz6gEVD", true},
		{"av", "https://www.bilibili.com/video/av117478344627500", "aid=117478344627500", true},
		{"av uppercase", "https://m.bilibili.com/video/AV42?p=2", "aid=42", true},
		{"short link is not a video url", "https://b23.tv/abc123", "", false},
		{"space", "https://space.bilibili.com/1", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, ok := parseVideoID(tt.url)
			if ok != tt.wantOK {
				t.Fatalf("parseVideoID(%q) ok = %v, want %v", tt.url, ok, tt.wantOK)
			}
			if ok && id.query() != tt.wantQuery {
				t.Errorf("query = %q, want %q", id.query(), tt.wantQuery)
			}
		})
	}
}

func TestParseResponse(t *testing.T) {
	const ok = `{"code":0,"message":"OK","data":{
		"bvid":"BV1xrnz6gEVD","title":"a title","desc":"a description",
		"pic":"http://i0.hdslb.com/bfs/archive/x.jpg",
		"owner":{"mid":546195,"name":"an author"}}}`
	const missing = `{"code":-404,"message":"not found","data":null}`

	t.Run("success", func(t *testing.T) {
		got, err := parseResponse(strings.NewReader(ok))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		checks := map[string][2]string{
			"Title":        {got.Title, "a title"},
			"AuthorName":   {got.AuthorName, "an author"},
			"AuthorURL":    {got.AuthorURL, "https://space.bilibili.com/546195"},
			"ThumbnailURL": {got.ThumbnailURL, "https://i0.hdslb.com/bfs/archive/x.jpg"},
			"HTML":         {got.HTML, "a description"},
			"URL":          {got.URL, "https://www.bilibili.com/video/BV1xrnz6gEVD"},
			"ProviderName": {got.ProviderName, "Bilibili"},
			"Type":         {got.Type, "rich"},
		}
		for field, c := range checks {
			if c[0] != c[1] {
				t.Errorf("%s = %q, want %q", field, c[0], c[1])
			}
		}
	})

	t.Run("api error carries code and message", func(t *testing.T) {
		_, err := parseResponse(strings.NewReader(missing))
		if err == nil {
			t.Fatal("expected error")
		}
		for _, want := range []string{"-404", "not found"} {
			if !strings.Contains(err.Error(), want) {
				t.Errorf("error %q does not contain %q", err, want)
			}
		}
	})

	t.Run("malformed json", func(t *testing.T) {
		if _, err := parseResponse(strings.NewReader("<html>")); err == nil {
			t.Fatal("expected error")
		}
	})
}
