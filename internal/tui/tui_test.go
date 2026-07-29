package tui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/TaylorJadin/site-archiving-toolkit/internal/config"
)

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
