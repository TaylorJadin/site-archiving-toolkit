package archive

import "testing"

func TestNormalizeURL(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"https://example.com", "example.com"},
		{"http://example.com/", "example.com"},
		{"https://example.com/path/to?q=1", "example.com-path-to-q=1"},
		{"https://a b.com", "a-b.com"},
	}
	for _, tt := range tests {
		if got := NormalizeURL(tt.in); got != tt.want {
			t.Errorf("NormalizeURL(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestValidURL(t *testing.T) {
	if !ValidURL("https://x.com") {
		t.Fatal("expected https valid")
	}
	if ValidURL("ftp://x.com") {
		t.Fatal("expected ftp invalid")
	}
}
