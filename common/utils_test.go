package common

import "testing"

func TestContainsOnlyAllowedReviewGIFURLs(t *testing.T) {
	for _, test := range []struct {
		comment string
		allowed bool
	}{
		{"https://media.tenor.com/example.gif", true},
		{"https://media.giphy.com/media/example/giphy.gif", true},
		{"https://static.klipy.com/example.webp", true},
		{"https://media.tenor.com/example.mp4", true},
		{"https://media.tenor.co/videos/example/mp4", true},
		{"https://tenor.com/view/okabe-rintaro-gif-25313171", false},
		{"https://klipy.com/gifs/caturday-happy-caturday-3", false},
		{"https://cdn.discordapp.com/attachments/example.gif", false},
		{"https://tenor.com/view/example", false},
		{"https://example.com/image.gif", false},
		{"example.com", false},
	} {
		if got := ContainsOnlyAllowedReviewGIFURLs(test.comment); got != test.allowed {
			t.Errorf("ContainsOnlyAllowedReviewGIFURLs(%q) = %t, want %t", test.comment, got, test.allowed)
		}
	}
}
