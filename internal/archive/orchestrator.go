package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CrawlStatus is the outcome of a single URL crawl.
type CrawlStatus string

const (
	StatusPending   CrawlStatus = "pending"
	StatusRunning   CrawlStatus = "running"
	StatusCompleted CrawlStatus = "completed"
	StatusSkipped   CrawlStatus = "skipped"
	StatusFailed    CrawlStatus = "failed"
	StatusCancelled CrawlStatus = "cancelled"
)

// Job is one URL in the archive queue.
type Job struct {
	URL           string
	NormalizedURL string
	Status        CrawlStatus
	Message       string
}

// EventType identifies orchestrator events.
type EventType int

const (
	EventLog EventType = iota
	EventJobStarted
	EventJobFinished
	EventBuildStarted
	EventBuildFinished
	EventAllDone
	EventError
)

// Event is a progress message from the orchestrator.
type Event struct {
	Type    EventType
	Index   int
	Status  CrawlStatus
	Message string
	Line    string
}

// Orchestrator runs a queue of crawl jobs.
type Orchestrator struct {
	cfg  *Config
	Jobs []Job

	// Events is closed when Run returns.
	Events chan Event

	current  *CrawlProcess
	skipCh   chan struct{}
	cancelCh chan struct{}
}

// NewOrchestrator rebuilds an orchestrator from a persisted session.
func NewOrchestrator(cfg *Config, session *Session) *Orchestrator {
	jobs := make([]Job, len(session.Jobs))
	copy(jobs, session.Jobs)
	for i := range jobs {
		if jobs[i].Status == StatusRunning {
			jobs[i].Status = StatusPending
		}
	}
	return &Orchestrator{
		cfg:      cfg,
		Jobs:     jobs,
		Events:   make(chan Event, 256),
		skipCh:   make(chan struct{}, 1),
		cancelCh: make(chan struct{}, 1),
	}
}

// SkipCurrent requests skipping the active crawl.
func (o *Orchestrator) SkipCurrent() {
	signal(o.skipCh)
	o.stopCurrent()
}

// CancelAll requests cancelling the entire archive run.
func (o *Orchestrator) CancelAll() {
	signal(o.cancelCh)
	o.stopCurrent()
}

func (o *Orchestrator) stopCurrent() {
	if o.current != nil {
		o.current.Stop()
	}
}

func (o *Orchestrator) emit(e Event) { o.Events <- e }

func (o *Orchestrator) log(line string) { o.emit(Event{Type: EventLog, Line: line}) }

func (o *Orchestrator) finish(i int, status CrawlStatus, message string) {
	o.Jobs[i].Status = status
	o.Jobs[i].Message = message
	o.emit(Event{Type: EventJobFinished, Index: i, Status: status, Message: message})
}

// Run executes the full archive pipeline and closes Events when it returns.
func (o *Orchestrator) Run(ctx context.Context) {
	defer close(o.Events)

	workdir := filepath.Join(o.cfg.RootDir, "crawls")
	if err := os.MkdirAll(workdir, 0o777); err != nil {
		o.emit(Event{Type: EventError, Message: err.Error()})
		return
	}

	o.emit(Event{Type: EventBuildStarted})
	o.log("Building Docker image: " + ImageName)
	if err := BuildImage(ctx, o.log); err != nil {
		if ctx.Err() != nil || o.cancelled() {
			o.markRemaining(0, StatusCancelled)
			o.emit(Event{Type: EventAllDone})
			return
		}
		o.emit(Event{Type: EventError, Message: fmt.Sprintf("docker build failed: %v", err)})
		return
	}
	o.emit(Event{Type: EventBuildFinished})

	for i := range o.Jobs {
		if o.cancelled() || ctx.Err() != nil {
			o.markRemaining(i, StatusCancelled)
			break
		}
		if !o.runJob(ctx, i, workdir) {
			break
		}
	}

	o.emit(Event{Type: EventAllDone})
}

// runJob archives o.Jobs[i]. It reports whether the queue should continue.
func (o *Orchestrator) runJob(ctx context.Context, i int, workdir string) bool {
	job := &o.Jobs[i]
	if job.Status != StatusPending {
		return true
	}

	if o.cfg.SkipExistingCrawls {
		if existing := findExistingCrawl(workdir, job.NormalizedURL, o.log); existing != "" {
			o.finish(i, StatusSkipped, "existing crawl: "+existing)
			return true
		}
	}

	now := time.Now().Format("2006-01-02T150405")
	crawlDir := filepath.Join(workdir, "INCOMPLETE-"+now+"-"+job.NormalizedURL)
	completeDir := filepath.Join(workdir, now+"-"+job.NormalizedURL)
	if err := os.MkdirAll(crawlDir, 0o777); err != nil {
		o.finish(i, StatusFailed, err.Error())
		return true
	}

	job.Status = StatusRunning
	o.emit(Event{Type: EventJobStarted, Index: i})
	o.log(fmt.Sprintf("Starting crawl %d/%d: %s", i+1, len(o.Jobs), job.URL))

	// Drop a skip request that arrived before this crawl started.
	drain(o.skipCh)

	proc := StartCrawl(ctx, RunOptions{
		CrawlDir:      crawlDir,
		URL:           job.URL,
		NormalizedURL: job.NormalizedURL,
		Timestamp:     now,
		Env:           o.cfg.crawlEnv(),
	}, o.log)
	o.current = proc

	waitCh := make(chan error, 1)
	go func() { waitCh <- proc.Wait() }()

	var runErr error
	stopped := ""
	select {
	case runErr = <-waitCh:
	case <-o.skipCh:
		stopped = "skip"
	case <-o.cancelCh:
		stopped = "cancel"
	case <-ctx.Done():
		stopped = "cancel"
	}
	if stopped != "" {
		proc.Stop()
		<-waitCh
	}
	o.current = nil

	// The container may have exited on its own just as a control arrived.
	switch {
	case stopped == "cancel" || o.cancelled() || ctx.Err() != nil:
		_ = os.RemoveAll(crawlDir)
		o.finish(i, StatusCancelled, "cancelled")
		o.markRemaining(i+1, StatusCancelled)
		return false
	case stopped == "skip" || o.consumeSkip():
		_ = os.RemoveAll(crawlDir)
		o.log("Skipped " + job.URL)
		o.finish(i, StatusSkipped, "skipped by user")
		return true
	case runErr != nil:
		_ = os.RemoveAll(crawlDir)
		o.finish(i, StatusFailed, runErr.Error())
		return true
	}

	if err := os.Rename(crawlDir, completeDir); err != nil {
		o.finish(i, StatusFailed, fmt.Sprintf("finalize crawl dir: %v", err))
		return true
	}
	o.log(fmt.Sprintf("Crawl of %s complete!", job.URL))
	o.finish(i, StatusCompleted, filepath.Base(completeDir))
	return true
}

// cancelled reports whether cancellation was requested, leaving the request in
// place so later checks still see it.
func (o *Orchestrator) cancelled() bool {
	select {
	case <-o.cancelCh:
		signal(o.cancelCh)
		return true
	default:
		return false
	}
}

func (o *Orchestrator) consumeSkip() bool { return drain(o.skipCh) }

func (o *Orchestrator) markRemaining(start int, status CrawlStatus) {
	for i := start; i < len(o.Jobs); i++ {
		if o.Jobs[i].Status == StatusPending || o.Jobs[i].Status == StatusRunning {
			o.Jobs[i].Status = status
			o.emit(Event{Type: EventJobFinished, Index: i, Status: status, Message: string(status)})
		}
	}
}

// findExistingCrawl returns the name of a completed crawl for normalizedURL,
// deleting any leftover incomplete crawls it finds along the way.
func findExistingCrawl(workdir, normalizedURL string, logFn func(string)) string {
	entries, err := os.ReadDir(workdir)
	if err != nil {
		return ""
	}
	suffix := "-" + normalizedURL
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || !strings.HasSuffix(name, suffix) {
			continue
		}
		if strings.HasPrefix(name, "INCOMPLETE-") {
			logFn("Removing incomplete crawl: " + name)
			_ = os.RemoveAll(filepath.Join(workdir, name))
			continue
		}
		return name
	}
	return ""
}

// signal makes a non-blocking request on a buffered channel of capacity 1.
func signal(ch chan struct{}) {
	select {
	case ch <- struct{}{}:
	default:
	}
}

// drain clears a pending request, reporting whether there was one.
func drain(ch chan struct{}) bool {
	select {
	case <-ch:
		return true
	default:
		return false
	}
}
