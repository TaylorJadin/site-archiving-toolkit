package archive

import (
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeURL(t *testing.T) {
	tests := []struct{ in, want string }{
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
	for _, u := range []string{"https://x.com", "http://x.com"} {
		if !ValidURL(u) {
			t.Errorf("expected %q valid", u)
		}
	}
	for _, u := range []string{"ftp://x.com", "x.com", ""} {
		if ValidURL(u) {
			t.Errorf("expected %q invalid", u)
		}
	}
}

func TestParseURLs(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{"newlines", "https://a.com\nhttps://b.com\nhttps://c.com/", []string{"https://a.com", "https://b.com", "https://c.com/"}},
		{"spaces", "https://a.com https://b.com", []string{"https://a.com", "https://b.com"}},
		{"mixed whitespace", "https://a.com\n  https://b.com \thttps://c.com/\n", []string{"https://a.com", "https://b.com", "https://c.com/"}},
		{"empty", "  \n\t ", []string{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ParseURLs(tc.raw); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("ParseURLs(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

func TestCollectURLsFromArgs(t *testing.T) {
	urls, background := CollectURLsFromArgs([]string{"https://a.com", "--background", "https://b.com https://c.com", "--other"})
	want := []string{"https://a.com", "https://b.com", "https://c.com"}
	if !reflect.DeepEqual(urls, want) {
		t.Errorf("urls = %v, want %v", urls, want)
	}
	if !background {
		t.Error("expected --background to be detected")
	}
}

func TestCollectURLsSkipsCommentsAndBlanks(t *testing.T) {
	got, err := collectURLs(strings.NewReader("# a comment\n\nhttps://a.com\n  \nhttps://b.com https://c.com\n"))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"https://a.com", "https://b.com", "https://c.com"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
