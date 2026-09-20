package bot

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestMakeURLKey(t *testing.T) {
	tests := []struct {
		name     string
		raw      string
		wantHost string
		wantPath string
		wantOK   bool
	}{
		{"plain", "https://www.instagram.com/p/abc", "instagram.com", "/p/abc", true},
		{"trailing slash dropped", "https://www.instagram.com/p/abc/", "instagram.com", "/p/abc", true},
		{"query dropped", "https://www.instagram.com/p/abc/?igsh=xyz123", "instagram.com", "/p/abc", true},
		{"fragment dropped", "https://www.threads.com/t/abc#c1", "threads.com", "/t/abc", true},
		{"www stripped", "https://instagram.com/p/abc", "instagram.com", "/p/abc", true},
		{"host lowercased", "HTTPS://WWW.Instagram.COM/p/abc", "instagram.com", "/p/abc", true},
		{"threads.net folds to threads.com", "https://www.threads.net/@u/post/abc", "threads.com", "/@u/post/abc", true},
		{"path case preserved", "https://www.instagram.com/p/AbC", "instagram.com", "/p/AbC", true},
		{"not a url", "hello world", "", "", false},
		{"no host", "https://", "", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := makeURLKey(tt.raw)
			if ok != tt.wantOK {
				t.Fatalf("makeURLKey(%q) ok = %v, want %v", tt.raw, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if got.host != tt.wantHost || got.path != tt.wantPath {
				t.Errorf("makeURLKey(%q) = {%q %q}, want {%q %q}",
					tt.raw, got.host, got.path, tt.wantHost, tt.wantPath)
			}
		})
	}
}

func TestMakeURLKeyEquivalence(t *testing.T) {
	a, okA := makeURLKey("https://www.instagram.com/p/abc/?igsh=xyz")
	b, okB := makeURLKey("https://instagram.com/p/abc")
	if !okA || !okB {
		t.Fatal("expected both URLs to parse")
	}
	if a != b {
		t.Errorf("keys differ: %+v vs %+v", a, b)
	}
}

func TestIsRichEmbed(t *testing.T) {
	tests := []struct {
		name  string
		embed *discordgo.MessageEmbed
		want  bool
	}{
		{"nil", nil, false},
		{"title only", &discordgo.MessageEmbed{Title: "Threads", URL: "https://x"}, false},
		{"description", &discordgo.MessageEmbed{Description: "a post"}, true},
		{"image", &discordgo.MessageEmbed{Image: &discordgo.MessageEmbedImage{URL: "https://i"}}, true},
		{"thumbnail", &discordgo.MessageEmbed{Thumbnail: &discordgo.MessageEmbedThumbnail{URL: "https://i"}}, true},
		{"video", &discordgo.MessageEmbed{Video: &discordgo.MessageEmbedVideo{URL: "https://v"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRichEmbed(tt.embed); got != tt.want {
				t.Errorf("isRichEmbed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasOfficialEmbed(t *testing.T) {
	const postURL = "https://www.instagram.com/p/abc/"

	rich := func(u string) *discordgo.MessageEmbed {
		return &discordgo.MessageEmbed{
			URL:       u,
			Title:     "a post",
			Type:      "link",
			Thumbnail: &discordgo.MessageEmbedThumbnail{URL: "https://i"},
		}
	}
	thin := func(u string) *discordgo.MessageEmbed {
		return &discordgo.MessageEmbed{URL: u, Title: "Instagram", Type: "link"}
	}
	msg := func(content string, embeds ...*discordgo.MessageEmbed) *discordgo.Message {
		return &discordgo.Message{Content: content, Embeds: embeds}
	}

	tests := []struct {
		name   string
		msg    *discordgo.Message
		rawURL string
		want   bool
	}{
		{"nil message", nil, postURL, false},
		{"no embeds", msg(postURL), postURL, false},
		{"rich embed same url", msg(postURL, rich(postURL)), postURL, true},
		{"rich embed query variant", msg(postURL, rich("https://www.instagram.com/p/abc")),
			"https://www.instagram.com/p/abc/?igsh=xyz", true},
		{"thin embed same url", msg(postURL, thin(postURL)), postURL, false},
		{"video type embed matches", msg(postURL, &discordgo.MessageEmbed{
			URL: postURL, Type: "video", Video: &discordgo.MessageEmbedVideo{URL: "https://v"},
		}), postURL, true},
		{"different host", msg(postURL, rich("https://www.threads.com/t/zzz")), postURL, false},
		{"embed without url ignored", msg(postURL, &discordgo.MessageEmbed{
			Thumbnail: &discordgo.MessageEmbedThumbnail{URL: "https://i"},
		}), postURL, false},
		{
			name:   "canonicalised share link, sole url on host",
			msg:    msg("https://www.instagram.com/share/xyz", rich("https://www.instagram.com/p/ABC/")),
			rawURL: "https://www.instagram.com/share/xyz",
			want:   true,
		},
		{
			name: "canonicalised path not credited when two urls share the host",
			msg: msg("https://www.instagram.com/share/xyz https://www.instagram.com/p/other/",
				rich("https://www.instagram.com/p/ABC/")),
			rawURL: "https://www.instagram.com/share/xyz",
			want:   false,
		},
		{
			name:   "thin embed does not trigger host fallback",
			msg:    msg("https://www.instagram.com/share/xyz", thin("https://www.instagram.com/p/ABC/")),
			rawURL: "https://www.instagram.com/share/xyz",
			want:   false,
		},
		{
			name: "matching embed among several",
			msg: msg("https://www.threads.com/t/aaa "+postURL,
				thin("https://www.threads.com/t/aaa"), rich(postURL)),
			rawURL: postURL,
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hasOfficialEmbed(tt.msg, tt.rawURL); got != tt.want {
				t.Errorf("hasOfficialEmbed(_, %q) = %v, want %v", tt.rawURL, got, tt.want)
			}
		})
	}
}

func TestPreviewsSuppressed(t *testing.T) {
	tests := []struct {
		name string
		msg  *discordgo.Message
		want bool
	}{
		{"nil", nil, false},
		{"no flags", &discordgo.Message{}, false},
		{"suppressed", &discordgo.Message{Flags: discordgo.MessageFlagsSuppressEmbeds}, true},
		{"suppressed among other flags", &discordgo.Message{
			Flags: discordgo.MessageFlagsSuppressNotifications | discordgo.MessageFlagsSuppressEmbeds,
		}, true},
		{"other flag only", &discordgo.Message{Flags: discordgo.MessageFlagsSuppressNotifications}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := previewsSuppressed(tt.msg); got != tt.want {
				t.Errorf("previewsSuppressed() = %v, want %v", got, tt.want)
			}
		})
	}
}
