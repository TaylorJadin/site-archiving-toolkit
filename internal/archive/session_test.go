package archive

import (
	"path/filepath"
	"testing"
)

func TestParseURLs(t *testing.T) {
	got := ParseURLs("https://a.com https://b.com\nhttps://c.com")
	want := []string{"https://a.com", "https://b.com", "https://c.com"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("%d: got %q want %q", i, got[i], want[i])
		}
	}
}

func TestSessionResumableURLs(t *testing.T) {
	s := &Session{
		Jobs: []Job{
			{URL: "https://a.com", Status: StatusCompleted},
			{URL: "https://b.com", Status: StatusCancelled},
			{URL: "https://c.com", Status: StatusSkipped},
		},
		Complete:  false,
		Resumable: true,
	}
	urls := s.ResumableURLs()
	if len(urls) != 1 || urls[0] != "https://b.com" {
		t.Fatalf("got %v", urls)
	}
}

func TestSessionSaveLoad(t *testing.T) {
	dir := t.TempDir()
	s := NewSession([]string{"https://example.com", "https://example.org"})
	if err := SaveSession(dir, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSession(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || len(loaded.Jobs) != 2 {
		t.Fatalf("loaded session: %+v", loaded)
	}
	if _, logPath, _ := SessionPaths(dir); filepath.Base(logPath) != ".last-session.log" {
		t.Fatalf("unexpected log path %s", logPath)
	}
}

func TestSessionApplyEventComplete(t *testing.T) {
	s := NewSession([]string{"https://example.com"})
	s.Jobs[0].Status = StatusCompleted
	s.ApplyEvent(Event{Type: EventAllDone})
	if !s.Complete || s.Resumable {
		t.Fatalf("expected complete non-resumable, got complete=%v resumable=%v", s.Complete, s.Resumable)
	}
}

func TestSessionApplyEventCancelled(t *testing.T) {
	s := NewSession([]string{"https://example.com", "https://example.org"})
	s.Jobs[0].Status = StatusCompleted
	s.Jobs[1].Status = StatusCancelled
	s.ApplyEvent(Event{Type: EventAllDone})
	if s.Complete || !s.Resumable {
		t.Fatalf("expected incomplete resumable, got complete=%v resumable=%v", s.Complete, s.Resumable)
	}
}
