package instagram

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"reels folds to reel", "https://www.instagram.com/reels/DdaejW2Jntf/", "https://www.instagram.com/reel/DdaejW2Jntf/"},
		{"reels with username", "https://www.instagram.com/someuser/reels/DdaejW2Jntf/", "https://www.instagram.com/someuser/reel/DdaejW2Jntf/"},
		{"reels without www", "https://instagram.com/reels/DdaejW2Jntf/", "https://instagram.com/reel/DdaejW2Jntf/"},
		{"reels with query", "https://www.instagram.com/reels/DdaejW2Jntf/?igsh=xyz", "https://www.instagram.com/reel/DdaejW2Jntf/?igsh=xyz"},
		{"reel left alone", "https://www.instagram.com/reel/DdaejW2Jntf/", "https://www.instagram.com/reel/DdaejW2Jntf/"},
		{"post left alone", "https://www.instagram.com/p/DdaejW2Jntf/", "https://www.instagram.com/p/DdaejW2Jntf/"},
		{"tv left alone", "https://www.instagram.com/tv/DdaejW2Jntf/", "https://www.instagram.com/tv/DdaejW2Jntf/"},
		{"share left alone", "https://www.instagram.com/share/DdaejW2Jntf/", "https://www.instagram.com/share/DdaejW2Jntf/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeURL(tt.raw); got != tt.want {
				t.Errorf("normalizeURL(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestExtractAuthorFromTitle(t *testing.T) {
	name, url := extractAuthor(
		"https://www.instagram.com/reel/DdaejW2Jntf/",
		"Brandon chacon (@the_paleo_feen) • Instagram reel",
	)
	if name != "@the_paleo_feen" || url != "https://www.instagram.com/the_paleo_feen" {
		t.Errorf("extractAuthor = (%q, %q), want (%q, %q)",
			name, url, "@the_paleo_feen", "https://www.instagram.com/the_paleo_feen")
	}
}
