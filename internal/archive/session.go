package archive

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

const (
	sessionFile   = ".last-session.json"
	sessionLog    = ".last-session.log"
	sessionControl = ".session.control"
)

// SessionPhase describes high-level crawl session state.
type SessionPhase string

const (
	SessionPhaseBuilding SessionPhase = "building"
	SessionPhaseCrawling SessionPhase = "crawling"
	SessionPhaseDone     SessionPhase = "done"
	SessionPhaseError    SessionPhase = "error"
	SessionPhaseCancelled SessionPhase = "cancelled"
)

// Session tracks the most recent archive run for reattach and resume.
type Session struct {
	Jobs         []Job       `json:"jobs"`
	CurrentIndex int         `json:"current_index"`
	Phase        SessionPhase `json:"phase"`
	Complete     bool        `json:"complete"`
	Resumable    bool        `json:"resumable"`
	Detached     bool        `json:"detached"`
	PID          int         `json:"pid"`
	StartedAt    time.Time   `json:"started_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
	Error        string      `json:"error,omitempty"`
}

// SessionPaths returns filesystem paths for session artifacts under crawls/.
func SessionPaths(rootDir string) (jsonPath, logPath, controlPath string) {
	dir := filepath.Join(rootDir, "crawls")
	return filepath.Join(dir, sessionFile),
		filepath.Join(dir, sessionLog),
		filepath.Join(dir, sessionControl)
}

// NewSession creates a session for a fresh URL list.
func NewSession(urls []string) *Session {
	now := time.Now().UTC()
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
	return &Session{
		Jobs:      jobs,
		Phase:     SessionPhaseBuilding,
		Resumable: false,
		StartedAt: now,
		UpdatedAt: now,
	}
}

// LoadSession reads the last session from disk.
func LoadSession(rootDir string) (*Session, error) {
	jsonPath, _, _ := SessionPaths(rootDir)
	data, err := os.ReadFile(jsonPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var s Session
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// SaveSession writes the session file.
func SaveSession(rootDir string, s *Session) error {
	if s == nil {
		return errors.New("nil session")
	}
	s.UpdatedAt = time.Now().UTC()
	jsonPath, _, _ := SessionPaths(rootDir)
	if err := os.MkdirAll(filepath.Dir(jsonPath), 0o777); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(jsonPath, data, 0o644)
}

// ClearSession removes persisted session artifacts.
func ClearSession(rootDir string) {
	jsonPath, logPath, controlPath := SessionPaths(rootDir)
	_ = os.Remove(jsonPath)
	_ = os.Remove(logPath)
	_ = os.Remove(controlPath)
}

// AppendSessionLog appends a line to the session log file.
func AppendSessionLog(rootDir, line string) error {
	_, logPath, _ := SessionPaths(rootDir)
	if err := os.MkdirAll(filepath.Dir(logPath), 0o777); err != nil {
		return err
	}
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, line)
	return err
}

// ReadSessionLog returns the full session log contents.
func ReadSessionLog(rootDir string) (string, error) {
	_, logPath, _ := SessionPaths(rootDir)
	data, err := os.ReadFile(logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}

// SendControl writes a control command for the session runner (skip/cancel).
func SendControl(rootDir, command string) error {
	_, _, controlPath := SessionPaths(rootDir)
	return os.WriteFile(controlPath, []byte(strings.TrimSpace(command)), 0o644)
}

// ConsumeControl reads and clears a pending control command, if any.
func ConsumeControl(rootDir string) (string, error) {
	_, _, controlPath := SessionPaths(rootDir)
	data, err := os.ReadFile(controlPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	_ = os.Remove(controlPath)
	return strings.TrimSpace(string(data)), nil
}

// ResumableURLs returns URLs that were not completed or skipped.
func (s *Session) ResumableURLs() []string {
	if s == nil {
		return nil
	}
	urls := make([]string, 0, len(s.Jobs))
	for _, j := range s.Jobs {
		if j.Status == StatusCompleted || j.Status == StatusSkipped {
			continue
		}
		urls = append(urls, j.URL)
	}
	return urls
}

// CanResume reports whether an incomplete session can be resumed.
func (s *Session) CanResume() bool {
	if s == nil || s.Complete || !s.Resumable {
		return false
	}
	return len(s.ResumableURLs()) > 0
}

// ProgressSummary returns a short human-readable progress string.
func (s *Session) ProgressSummary() string {
	if s == nil || len(s.Jobs) == 0 {
		return ""
	}
	done := 0
	for _, j := range s.Jobs {
		switch j.Status {
		case StatusCompleted, StatusSkipped:
			done++
		}
	}
	return fmt.Sprintf("%d/%d complete", done, len(s.Jobs))
}

// IsRunnerActive reports whether the session runner process appears alive.
func IsRunnerActive(rootDir string) (bool, *Session) {
	s, err := LoadSession(rootDir)
	if err != nil || s == nil || s.Complete {
		return false, s
	}
	if s.PID > 0 && processAlive(s.PID) {
		return true, s
	}
	running, err := IsCrawlRunning()
	if err == nil && running {
		return true, s
	}
	return false, s
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	return err == nil
}

// ApplyEvent updates session state from an orchestrator event.
func (s *Session) ApplyEvent(ev Event) {
	if s == nil {
		return
	}
	switch ev.Type {
	case EventBuildStarted:
		s.Phase = SessionPhaseBuilding
	case EventBuildFinished:
		s.Phase = SessionPhaseCrawling
	case EventJobStarted:
		s.Phase = SessionPhaseCrawling
		s.CurrentIndex = ev.Index
	case EventJobFinished:
		if ev.Index >= 0 && ev.Index < len(s.Jobs) {
			s.Jobs[ev.Index].Status = ev.Status
			s.Jobs[ev.Index].Message = ev.Message
		}
	case EventError:
		s.Phase = SessionPhaseError
		s.Error = ev.Message
		s.Complete = true
		s.Resumable = true
	case EventAllDone:
		s.Resumable = false
		s.Complete = true
		s.Phase = SessionPhaseDone
		for _, j := range s.Jobs {
			if j.Status != StatusCompleted && j.Status != StatusSkipped {
				s.Complete = false
				s.Resumable = true
				if j.Status == StatusCancelled {
					s.Phase = SessionPhaseCancelled
				} else if s.Phase == SessionPhaseDone {
					s.Phase = SessionPhaseCrawling
				}
				break
			}
		}
	}
}
