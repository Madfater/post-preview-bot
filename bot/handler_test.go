package bot

import (
	"reflect"
	"testing"
)

func TestExtractURLs(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []string
	}{
		{
			name:    "no urls",
			content: "just some text",
			want:    []string{},
		},
		{
			name:    "plain url",
			content: "look at https://www.threads.com/@user/post/abc123",
			want:    []string{"https://www.threads.com/@user/post/abc123"},
		},
		{
			name:    "trailing period trimmed",
			content: "see https://www.threads.com/@user/post/abc123.",
			want:    []string{"https://www.threads.com/@user/post/abc123"},
		},
		{
			name:    "trailing paren trimmed",
			content: "(https://www.instagram.com/p/abc123/)",
			want:    []string{"https://www.instagram.com/p/abc123/"},
		},
		{
			name:    "angle wrapped url excluded",
			content: "quiet please <https://www.threads.com/@user/post/abc123>",
			want:    []string{},
		},
		{
			name:    "masked link excluded",
			content: "[my post](https://www.instagram.com/p/abc123/)",
			want:    []string{},
		},
		{
			name:    "wrapped and unwrapped mixed",
			content: "<https://www.threads.com/t/aaa> and https://www.instagram.com/p/bbb/",
			want:    []string{"https://www.instagram.com/p/bbb/"},
		},
		{
			name:    "multiple urls",
			content: "https://www.threads.com/t/aaa https://www.instagram.com/reel/bbb/",
			want: []string{
				"https://www.threads.com/t/aaa",
				"https://www.instagram.com/reel/bbb/",
			},
		},
		{
			name:    "duplicates collapsed",
			content: "https://www.threads.com/t/aaa again https://www.threads.com/t/aaa",
			want:    []string{"https://www.threads.com/t/aaa"},
		},
		{
			name:    "query string preserved",
			content: "https://www.instagram.com/p/abc/?igsh=xyz123",
			want:    []string{"https://www.instagram.com/p/abc/?igsh=xyz123"},
		},
		{
			name:    "two angle wrapped urls",
			content: "<https://www.threads.com/t/aaa> <https://www.threads.com/t/bbb>",
			want:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractURLs(tt.content)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("extractURLs(%q) = %#v, want %#v", tt.content, got, tt.want)
			}
		})
	}
}
