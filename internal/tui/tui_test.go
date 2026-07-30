package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/TaylorJadin/site-archiving-toolkit/internal/archive"
	"github.com/TaylorJadin/site-archiving-toolkit/internal/config"
)

func TestParseURLsSpaceAndNewline(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "newlines",
			raw:  "https://example.com\nhttps://another-site.org\nhttps://third.example/",
			want: []string{"https://example.com", "https://another-site.org", "https://third.example/"},
		},
		{
			name: "spaces",
			raw:  "https://example.com https://another-site.org https://third.example/",
			want: []string{"https://example.com", "https://another-site.org", "https://third.example/"},
		},
		{
			name: "mixed whitespace",
			raw:  "https://example.com\n  https://another-site.org \thttps://third.example/\n",
			want: []string{"https://example.com", "https://another-site.org", "https://third.example/"},
		},
		{
			name: "empty",
			raw:  "  \n\t ",
			want: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := archive.ParseURLs(tc.raw)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d urls %v, want %d %v", len(got), got, len(tc.want), tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("url %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestPasteMultilineURLs(t *testing.T) {
	m := New(&config.Config{}, Options{})
	pasted := "https://example.com\nhttps://another-site.org\nhttps://third.example/"

	next, _ := m.Update(tea.PasteMsg{Content: pasted})
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", next)
	}

	value := got.textarea.Value()
	lines := strings.Split(strings.TrimRight(value, "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 URL lines after paste, got %d: %q", len(lines), value)
	}
}

func TestViewInputHasNoBoxBorder(t *testing.T) {
	m := New(&config.Config{}, Options{})
	m.width, m.height = 80, 24
	m.resize()
	view := m.viewInput()
	if !strings.Contains(view, "Enter URL(s) to crawl (multiple URLs can be separated by a space or new line)") {
		t.Fatalf("missing updated subtitle in view:\n%s", view)
	}
	for _, ch := range []string{"╭", "╮", "╰", "╯"} {
		if strings.Contains(view, ch) {
			t.Fatalf("view still has boxed border corner %q:\n%s", ch, view)
		}
	}
}

func TestViewDoesNotUseAltScreen(t *testing.T) {
	m := New(&config.Config{}, Options{})
	v := m.View()
	if v.AltScreen {
		t.Fatal("expected alt-screen disabled")
	}
}

func TestViewRunningKeyHints(t *testing.T) {
	m := New(&config.Config{}, Options{})
	m.phase = phaseRunning
	m.building = false
	m.width, m.height = 80, 24
	m.resize()
	view := m.viewRunning()
	for _, want := range []string{"s skip url", "d detach", "q quit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in running view:\n%s", want, view)
		}
	}
	for _, ban := range []string{"c cancel", "esc stop"} {
		if strings.Contains(view, ban) {
			t.Fatalf("unexpected leftover hint %q in running view:\n%s", ban, view)
		}
	}
}
