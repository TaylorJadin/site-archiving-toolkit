package archive

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/TaylorJadin/site-archiving-toolkit/internal/config"
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

// EventType identifies orchestrator events sent to the TUI.
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

// Event is a message from the crawler to the UI.
type Event struct {
	Type    EventType
	Index   int
	Total   int
	URL     string
	Status  CrawlStatus
	Message string
	Line    string
}

// Orchestrator runs a queue of crawl jobs.
type Orchestrator struct {
	Cfg    *config.Config
	Jobs   []Job
	Events chan Event

	current  *CrawlProcess
	skipCh   chan struct{}
	cancelCh chan struct{}
}

// NewOrchestrator creates an orchestrator for the given URLs.
func NewOrchestrator(cfg *config.Config, urls []string) *Orchestrator {
	jobs := make([]Job, 0, len(urls))
	for _, u := range urls {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		jobs = append(jobs, Job{
			URL:           u,
			NormalizedURL: NormalizeURL(u),
			Status:        StatusPending,
		})
	}
	return &Orchestrator{
		Cfg:      cfg,
		Jobs:     jobs,
		Events:   make(chan Event, 256),
		skipCh:   make(chan struct{}, 1),
		cancelCh: make(chan struct{}, 1),
	}
}

// NewOrchestratorFromSession rebuilds an orchestrator from a persisted session.
func NewOrchestratorFromSession(cfg *config.Config, session *Session) *Orchestrator {
	jobs := make([]Job, len(session.Jobs))
	copy(jobs, session.Jobs)
	for i := range jobs {
		if jobs[i].Status == StatusRunning {
			jobs[i].Status = StatusPending
		}
	}
	return &Orchestrator{
		Cfg:      cfg,
		Jobs:     jobs,
		Events:   make(chan Event, 256),
		skipCh:   make(chan struct{}, 1),
		cancelCh: make(chan struct{}, 1),
	}
}

// SkipCurrent requests skipping the active crawl.
func (o *Orchestrator) SkipCurrent() {
	select {
	case o.skipCh <- struct{}{}:
	default:
	}
	o.stopCurrent()
}

// CancelAll requests cancelling the entire archive run.
func (o *Orchestrator) CancelAll() {
	select {
	case o.cancelCh <- struct{}{}:
	default:
	}
	o.stopCurrent()
}

func (o *Orchestrator) stopCurrent() {
	if o.current != nil {
		o.current.Stop()
	}
}

func (o *Orchestrator) emit(e Event) {
	select {
	case o.Events <- e:
	default:
		if e.Type != EventLog {
			o.Events <- e
		}
	}
}

func (o *Orchestrator) log(line string) {
	o.emit(Event{Type: EventLog, Line: line})
}

// Run executes the full archive pipeline.
func (o *Orchestrator) Run(ctx context.Context) {
	defer close(o.Events)

	if len(o.Jobs) == 0 {
		o.emit(Event{Type: EventError, Message: "no URLs to archive"})
		return
	}

	for _, j := range o.Jobs {
		if !ValidURL(j.URL) {
			o.emit(Event{Type: EventError, Message: fmt.Sprintf("URL must start with http:// or https://: %s", j.URL)})
			return
		}
	}

	if err := DockerAvailable(); err != nil {
		o.emit(Event{Type: EventError, Message: err.Error()})
		return
	}

	running, err := IsCrawlRunning()
	if err != nil {
		o.emit(Event{Type: EventError, Message: err.Error()})
		return
	}
	if running {
		o.emit(Event{Type: EventError, Message: "a crawl is already running; reattach with archive or use 'archive quit' first"})
		return
	}

	workdir := filepath.Join(o.Cfg.RootDir, "crawls")
	if err := os.MkdirAll(workdir, 0o777); err != nil {
		o.emit(Event{Type: EventError, Message: err.Error()})
		return
	}

	iniPath := filepath.Join(o.Cfg.RootDir, "archive.ini")
	if err := o.Cfg.WriteArchiveINI(iniPath); err != nil {
		o.emit(Event{Type: EventError, Message: err.Error()})
		return
	}

	o.emit(Event{Type: EventBuildStarted, Message: "Building webrecorder Docker image..."})
	o.log("Building Docker image: " + ImageName)
	if err := BuildImage(ctx, o.Cfg.RootDir, o.log); err != nil {
		if ctx.Err() != nil || o.cancelled() {
			o.markFrom(0, StatusCancelled)
			o.emit(Event{Type: EventAllDone})
			return
		}
		o.emit(Event{Type: EventError, Message: fmt.Sprintf("docker build failed: %v", err)})
		return
	}
	o.emit(Event{Type: EventBuildFinished, Message: "Image ready"})

	total := len(o.Jobs)
	for i := range o.Jobs {
		if o.cancelled() || ctx.Err() != nil {
			o.markFrom(i, StatusCancelled)
			break
		}

		job := &o.Jobs[i]
		if job.Status == StatusCompleted || job.Status == StatusSkipped {
			continue
		}
		if job.Status == StatusCancelled {
			continue
		}

		if o.Cfg.SkipExistingCrawls {
			if skipped, name := shouldSkipExisting(workdir, job.NormalizedURL, o.log); skipped {
				job.Status = StatusSkipped
				job.Message = "existing crawl: " + name
				o.emit(Event{
					Type:    EventJobFinished,
					Index:   i,
					Total:   total,
					URL:     job.URL,
					Status:  StatusSkipped,
					Message: job.Message,
				})
				continue
			}
		}

		now := time.Now().Format("2006-01-02T150405")
		crawlDir := filepath.Join(workdir, "INCOMPLETE-"+now+"-"+job.NormalizedURL)
		completeDir := filepath.Join(workdir, now+"-"+job.NormalizedURL)

		if err := os.MkdirAll(crawlDir, 0o777); err != nil {
			job.Status = StatusFailed
			job.Message = err.Error()
			o.emit(Event{Type: EventJobFinished, Index: i, Total: total, URL: job.URL, Status: StatusFailed, Message: job.Message})
			continue
		}

		job.Status = StatusRunning
		o.emit(Event{Type: EventJobStarted, Index: i, Total: total, URL: job.URL})
		o.log(fmt.Sprintf("Starting crawl %d/%d: %s", i+1, total, job.URL))

		select {
		case <-o.skipCh:
		default:
		}

		proc, err := StartCrawl(ctx, RunOptions{
			RootDir:       o.Cfg.RootDir,
			CrawlDir:      crawlDir,
			URL:           job.URL,
			NormalizedURL: job.NormalizedURL,
			Timestamp:     now,
			ArchiveINI:    iniPath,
		}, o.log)
		if err != nil {
			_ = os.RemoveAll(crawlDir)
			job.Status = StatusFailed
			job.Message = err.Error()
			o.emit(Event{Type: EventJobFinished, Index: i, Total: total, URL: job.URL, Status: StatusFailed, Message: job.Message})
			continue
		}
		o.current = proc

		waitErr := make(chan error, 1)
		go func() { waitErr <- proc.Wait() }()

		outcome := "ok"
		select {
		case err := <-waitErr:
			if o.cancelled() {
				outcome = "cancelled"
			} else if o.consumeSkip() {
				outcome = "skipped"
			} else if err != nil {
				if o.cancelled() {
					outcome = "cancelled"
				} else if o.consumeSkip() {
					outcome = "skipped"
				} else {
					outcome = "failed"
					job.Status = StatusFailed
					job.Message = err.Error()
					_ = os.RemoveAll(crawlDir)
					o.current = nil
					o.emit(Event{Type: EventJobFinished, Index: i, Total: total, URL: job.URL, Status: StatusFailed, Message: job.Message})
					continue
				}
			}
		case <-o.skipCh:
			outcome = "skipped"
			proc.Stop()
			<-waitErr
		case <-o.cancelCh:
			outcome = "cancelled"
			proc.Stop()
			<-waitErr
		case <-ctx.Done():
			outcome = "cancelled"
			proc.Stop()
			<-waitErr
		}
		o.current = nil

		switch outcome {
		case "cancelled":
			job.Status = StatusCancelled
			job.Message = "cancelled"
			_ = os.RemoveAll(crawlDir)
			o.emit(Event{Type: EventJobFinished, Index: i, Total: total, URL: job.URL, Status: StatusCancelled, Message: job.Message})
			o.markFrom(i+1, StatusCancelled)
			o.emit(Event{Type: EventAllDone})
			return
		case "skipped":
			job.Status = StatusSkipped
			job.Message = "skipped by user"
			_ = os.RemoveAll(crawlDir)
			o.log(fmt.Sprintf("Skipped %s", job.URL))
			o.emit(Event{Type: EventJobFinished, Index: i, Total: total, URL: job.URL, Status: StatusSkipped, Message: job.Message})
			continue
		}

		if err := os.Rename(crawlDir, completeDir); err != nil {
			job.Status = StatusFailed
			job.Message = fmt.Sprintf("finalize crawl dir: %v", err)
			o.emit(Event{Type: EventJobFinished, Index: i, Total: total, URL: job.URL, Status: StatusFailed, Message: job.Message})
			continue
		}

		job.Status = StatusCompleted
		job.Message = filepath.Base(completeDir)
		o.log(fmt.Sprintf("Crawl of %s complete!", job.URL))
		o.emit(Event{Type: EventJobFinished, Index: i, Total: total, URL: job.URL, Status: StatusCompleted, Message: job.Message})
	}

	o.emit(Event{Type: EventAllDone})
}

func (o *Orchestrator) cancelled() bool {
	select {
	case <-o.cancelCh:
		select {
		case o.cancelCh <- struct{}{}:
		default:
		}
		return true
	default:
		return false
	}
}

func (o *Orchestrator) consumeSkip() bool {
	select {
	case <-o.skipCh:
		return true
	default:
		return false
	}
}

func (o *Orchestrator) markFrom(start int, status CrawlStatus) {
	for i := start; i < len(o.Jobs); i++ {
		if o.Jobs[i].Status == StatusPending || o.Jobs[i].Status == StatusRunning {
			o.Jobs[i].Status = status
		}
	}
}

func shouldSkipExisting(workdir, normalizedURL string, logFn func(string)) (bool, string) {
	entries, err := os.ReadDir(workdir)
	if err != nil {
		return false, ""
	}

	suffix := "-" + normalizedURL
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		if strings.HasPrefix(name, "INCOMPLETE-") {
			logFn("Removing incomplete crawl: " + name)
			_ = os.RemoveAll(filepath.Join(workdir, name))
			continue
		}
		return true, name
	}
	return false, ""
}
