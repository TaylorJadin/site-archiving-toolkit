package archive

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestFindExistingCrawl(t *testing.T) {
	workdir := t.TempDir()
	for _, name := range []string{
		"2026-01-01T000000-example.com",
		"INCOMPLETE-2026-01-02T000000-example.org",
		"2026-01-01T000000-other.com",
	} {
		if err := os.Mkdir(filepath.Join(workdir, name), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(workdir, "notes-example.net"), nil, 0o644); err != nil {
		t.Fatal(err)
	}

	var logs []string
	log := func(line string) { logs = append(logs, line) }

	if got := findExistingCrawl(workdir, "example.com", log); got != "2026-01-01T000000-example.com" {
		t.Errorf("completed crawl not found, got %q", got)
	}
	// An incomplete crawl is not a reason to skip; it is cleaned up instead.
	if got := findExistingCrawl(workdir, "example.org", log); got != "" {
		t.Errorf("incomplete crawl should not count as existing, got %q", got)
	}
	if _, err := os.Stat(filepath.Join(workdir, "INCOMPLETE-2026-01-02T000000-example.org")); !os.IsNotExist(err) {
		t.Error("incomplete crawl was not removed")
	}
	if len(logs) != 1 {
		t.Errorf("expected one log line about the removal, got %v", logs)
	}
	// A file, not a directory, must not match.
	if got := findExistingCrawl(workdir, "example.net", log); got != "" {
		t.Errorf("expected no match for a plain file, got %q", got)
	}
	if got := findExistingCrawl(workdir, "unseen.com", log); got != "" {
		t.Errorf("expected no match, got %q", got)
	}
}

func TestSignalAndDrain(t *testing.T) {
	ch := make(chan struct{}, 1)
	if drain(ch) {
		t.Error("empty channel should not report a pending request")
	}
	signal(ch)
	signal(ch) // must not block when a request is already pending
	if !drain(ch) {
		t.Error("expected a pending request")
	}
	if drain(ch) {
		t.Error("request should have been consumed")
	}
}

func TestCancelledLeavesRequestInPlace(t *testing.T) {
	o := NewOrchestrator(&Config{}, NewSession([]string{"https://a.com"}))
	if o.cancelled() {
		t.Fatal("no cancellation was requested")
	}
	signal(o.cancelCh)
	for i := range 3 {
		if !o.cancelled() {
			t.Fatalf("cancellation lost after %d checks", i)
		}
	}
}

// Run swaps the current process while the control watcher may be stopping it.
func TestStopCurrentIsConcurrencySafe(t *testing.T) {
	o := NewOrchestrator(&Config{}, NewSession([]string{"https://a.com"}))

	// Stop shells out to docker, so hand out processes that are already spent.
	spent := func() *CrawlProcess {
		p := &CrawlProcess{cancel: func() {}, done: make(chan error, 1)}
		p.once.Do(func() {})
		return p
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for range 1000 {
			o.setCurrent(spent())
			o.setCurrent(nil)
		}
	}()
	go func() {
		defer wg.Done()
		for range 1000 {
			o.stopCurrent()
		}
	}()
	wg.Wait()
}

func TestNewOrchestratorResetsInterruptedJobs(t *testing.T) {
	session := &Session{Jobs: []Job{
		{URL: "https://a.com", Status: StatusRunning},
		{URL: "https://b.com", Status: StatusCompleted},
	}}
	o := NewOrchestrator(&Config{}, session)

	if o.Jobs[0].Status != StatusPending {
		t.Errorf("interrupted job status = %q, want pending", o.Jobs[0].Status)
	}
	if o.Jobs[1].Status != StatusCompleted {
		t.Errorf("finished job status = %q, want completed", o.Jobs[1].Status)
	}
	// The orchestrator must work on a copy so the session is not mutated.
	if session.Jobs[0].Status != StatusRunning {
		t.Error("orchestrator mutated the session it was built from")
	}
}
