package archive

import (
	"os"
	"reflect"
	"testing"
)

func TestSessionResumableURLs(t *testing.T) {
	s := &Session{Jobs: []Job{
		{URL: "https://a.com", Status: StatusCompleted},
		{URL: "https://b.com", Status: StatusCancelled},
		{URL: "https://c.com", Status: StatusSkipped},
		{URL: "https://d.com", Status: StatusFailed},
	}}
	want := []string{"https://b.com", "https://d.com"}
	if got := s.ResumableURLs(); !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	if !s.CanResume() {
		t.Fatal("expected session to be resumable")
	}
	if got, want := s.ProgressSummary(), "2/4 complete"; got != want {
		t.Fatalf("ProgressSummary() = %q, want %q", got, want)
	}
}

func TestSessionSaveLoad(t *testing.T) {
	dir := t.TempDir()

	if s, err := LoadSession(dir); err != nil || s != nil {
		t.Fatalf("LoadSession on empty dir = (%v, %v), want (nil, nil)", s, err)
	}

	s := NewSession([]string{"https://example.com", " ", "https://example.org"})
	if len(s.Jobs) != 2 {
		t.Fatalf("expected blank URLs to be dropped, got %d jobs", len(s.Jobs))
	}
	if err := SaveSession(dir, s); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadSession(dir)
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || len(loaded.Jobs) != 2 || loaded.Jobs[0].NormalizedURL != "example.com" {
		t.Fatalf("loaded session: %+v", loaded)
	}
	if _, err := os.Stat(sessionJSONPath(dir)); err != nil {
		t.Fatalf("session file not written: %v", err)
	}
}

func TestSessionLogAndControl(t *testing.T) {
	dir := t.TempDir()
	if err := SaveSession(dir, NewSession([]string{"https://example.com"})); err != nil {
		t.Fatal(err)
	}

	for _, line := range []string{"first", "second"} {
		if err := AppendSessionLog(dir, line); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := ReadSessionLog(dir); err != nil || got != "first\nsecond\n" {
		t.Fatalf("ReadSessionLog() = (%q, %v)", got, err)
	}

	if cmd, err := ConsumeControl(dir); err != nil || cmd != "" {
		t.Fatalf("ConsumeControl() with no command = (%q, %v)", cmd, err)
	}
	if err := SendControl(dir, "cancel"); err != nil {
		t.Fatal(err)
	}
	if cmd, err := ConsumeControl(dir); err != nil || cmd != "cancel" {
		t.Fatalf("ConsumeControl() = (%q, %v), want cancel", cmd, err)
	}
	if cmd, _ := ConsumeControl(dir); cmd != "" {
		t.Fatalf("control command was not cleared, got %q", cmd)
	}
}

func TestSessionApplyEventAllDone(t *testing.T) {
	cases := []struct {
		name      string
		statuses  []CrawlStatus
		wantPhase SessionPhase
		wantAgain bool
	}{
		{"all completed", []CrawlStatus{StatusCompleted}, SessionPhaseDone, false},
		{"cancelled", []CrawlStatus{StatusCompleted, StatusCancelled}, SessionPhaseCancelled, true},
		{"failed", []CrawlStatus{StatusFailed}, SessionPhaseDone, true},
		{"skipped", []CrawlStatus{StatusSkipped}, SessionPhaseDone, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := &Session{Jobs: make([]Job, len(tc.statuses))}
			for i, st := range tc.statuses {
				s.Jobs[i] = Job{URL: "https://example.com", Status: st}
			}
			s.ApplyEvent(Event{Type: EventAllDone})

			// A finished runner always leaves the session complete, otherwise
			// anything waiting on it (the TUI, RunHeadless) would hang forever.
			if !s.Complete {
				t.Error("expected Complete after EventAllDone")
			}
			if s.Phase != tc.wantPhase {
				t.Errorf("Phase = %q, want %q", s.Phase, tc.wantPhase)
			}
			if got := s.CanResume(); got != tc.wantAgain {
				t.Errorf("CanResume() = %v, want %v", got, tc.wantAgain)
			}
		})
	}
}

func TestSessionApplyEventError(t *testing.T) {
	s := NewSession([]string{"https://example.com"})
	s.ApplyEvent(Event{Type: EventError, Message: "docker build failed"})
	if !s.Complete || s.Phase != SessionPhaseError || s.Error != "docker build failed" {
		t.Fatalf("unexpected session after error: %+v", s)
	}
	if !s.CanResume() {
		t.Fatal("expected a failed session to be resumable")
	}
}

func TestSessionApplyEventJobs(t *testing.T) {
	s := NewSession([]string{"https://a.com", "https://b.com"})
	s.ApplyEvent(Event{Type: EventJobStarted, Index: 1})
	if s.CurrentIndex != 1 || s.Phase != SessionPhaseCrawling {
		t.Fatalf("unexpected session after job start: %+v", s)
	}
	s.ApplyEvent(Event{Type: EventJobFinished, Index: 1, Status: StatusCompleted, Message: "dir"})
	if s.Jobs[1].Status != StatusCompleted || s.Jobs[1].Message != "dir" {
		t.Fatalf("job not updated: %+v", s.Jobs[1])
	}
	// Out-of-range indexes must not panic.
	s.ApplyEvent(Event{Type: EventJobFinished, Index: 9, Status: StatusFailed})
}

func TestRunnerAliveAndStopped(t *testing.T) {
	unknown := &Session{PID: 0}
	if unknown.RunnerAlive() || unknown.RunnerStopped() {
		t.Fatal("a session without a PID is neither alive nor stopped")
	}
	self := &Session{PID: os.Getpid()}
	if !self.RunnerAlive() || self.RunnerStopped() {
		t.Fatal("expected the current process to look alive")
	}
}
