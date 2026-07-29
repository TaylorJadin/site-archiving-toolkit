package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

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
			got := parseURLs(tc.raw)
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
	m := New(&config.Config{})
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
	want := []string{
		"https://example.com",
		"https://another-site.org",
		"https://third.example/",
	}
	for i, line := range want {
		if lines[i] != line {
			t.Errorf("line %d: got %q, want %q", i, lines[i], line)
		}
	}
}

func TestEnterStartsWithoutInsertingNewline(t *testing.T) {
	m := New(&config.Config{})
	m.textarea.SetValue("https://example.com")

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	got, ok := next.(Model)
	if !ok {
		t.Fatalf("expected Model, got %T", next)
	}
	if got.phase != phaseRunning && got.errMsg == "" {
		// Without a real config/root, start may error — either running or a
		// validation/setup error is fine; the key is that Enter did not stay
		// on input with an extra blank line.
		if got.phase == phaseInput && strings.Contains(got.textarea.Value(), "\n") {
			t.Fatalf("enter inserted a newline instead of starting: %q", got.textarea.Value())
		}
	}
	if got.phase == phaseInput && got.errMsg == "" && got.textarea.Value() != "https://example.com" {
		t.Fatalf("unexpected textarea after enter: %q (phase=%v err=%q)", got.textarea.Value(), got.phase, got.errMsg)
	}
}

func TestViewInputHasNoBoxBorder(t *testing.T) {
	m := New(&config.Config{})
	m.width, m.height = 80, 24
	m.resize()
	view := m.viewInput()
	if !strings.Contains(view, "Enter URL(s) to crawl (multiple URLs can be separated by a space or new line)") {
		t.Fatalf("missing updated subtitle in view:\n%s", view)
	}
	// Rounded box borders use these lipgloss characters; the bare textarea should not.
	for _, ch := range []string{"╭", "╮", "╰", "╯"} {
		if strings.Contains(view, ch) {
			t.Fatalf("view still has boxed border corner %q:\n%s", ch, view)
		}
	}
}
