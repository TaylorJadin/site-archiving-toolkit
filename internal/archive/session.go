package archive

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionPhase describes high-level crawl session state.
type SessionPhase string

const (
	SessionPhaseBuilding  SessionPhase = "building"
	SessionPhaseCrawling  SessionPhase = "crawling"
	SessionPhaseDone      SessionPhase = "done"
	SessionPhaseCancelled SessionPhase = "cancelled"
	SessionPhaseError     SessionPhase = "error"
)

// Session tracks the most recent archive run so the TUI can reattach or resume.
// Complete means the runner process finished, whatever the outcome; Phase says
// how it finished.
type Session struct {
	Jobs         []Job        `json:"jobs"`
	CurrentIndex int          `json:"current_index"`
	Phase        SessionPhase `json:"phase"`
	Complete     bool         `json:"complete"`
	PID          int          `json:"pid"`
	StartedAt    time.Time    `json:"started_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Error        string       `json:"error,omitempty"`
}

// sessionPath returns the path of a session artifact under crawls/.
func sessionPath(rootDir, name string) string {
	return filepath.Join(rootDir, "crawls", name)
}

func sessionJSONPath(rootDir string) string    { return sessionPath(rootDir, ".last-session.json") }
func sessionLogPath(rootDir string) string     { return sessionPath(rootDir, ".last-session.log") }
func sessionControlPath(rootDir string) string { return sessionPath(rootDir, ".session.control") }

// NewSession creates a session for a fresh URL list.
func NewSession(urls []string) *Session {
	now := time.Now().UTC()
	jobs := make([]Job, 0, len(urls))
	for _, u := range urls {
		if u = strings.TrimSpace(u); u == "" {
			continue
		}
		jobs = append(jobs, Job{URL: u, NormalizedURL: NormalizeURL(u), Status: StatusPending})
	}
	return &Session{
		Jobs:         jobs,
		CurrentIndex: -1,
		Phase:        SessionPhaseBuilding,
		StartedAt:    now,
		UpdatedAt:    now,
	}
}

// LoadSession reads the last session from disk. It returns (nil, nil) when no
// session has been recorded yet.
func LoadSession(rootDir string) (*Session, error) {
	data, err := os.ReadFile(sessionJSONPath(rootDir))
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

// SaveSession writes the session file. The write goes through a temporary file
// so a reader polling the session never sees a half-written one.
func SaveSession(rootDir string, s *Session) error {
	if s == nil {
		return errors.New("nil session")
	}
	s.UpdatedAt = time.Now().UTC()
	path := sessionJSONPath(rootDir)
	if err := os.MkdirAll(filepath.Dir(path), 0o777); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// AppendSessionLog appends a line to the session log file.
func AppendSessionLog(rootDir, line string) error {
	f, err := os.OpenFile(sessionLogPath(rootDir), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = fmt.Fprintln(f, line)
	return err
}

// ReadSessionLog returns the full session log contents.
func ReadSessionLog(rootDir string) (string, error) {
	data, err := os.ReadFile(sessionLogPath(rootDir))
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
	return os.WriteFile(sessionControlPath(rootDir), []byte(command), 0o644)
}

// ConsumeControl reads and clears a pending control command, if any.
func ConsumeControl(rootDir string) (string, error) {
	path := sessionControlPath(rootDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	_ = os.Remove(path)
	return strings.TrimSpace(string(data)), nil
}

// ResumableURLs returns URLs that were neither completed nor skipped.
func (s *Session) ResumableURLs() []string {
	if s == nil {
		return nil
	}
	var urls []string
	for _, j := range s.Jobs {
		if j.Status != StatusCompleted && j.Status != StatusSkipped {
			urls = append(urls, j.URL)
		}
	}
	return urls
}

// CanResume reports whether the session left any URL unarchived.
func (s *Session) CanResume() bool {
	return len(s.ResumableURLs()) > 0
}

// ProgressSummary returns a short human-readable progress string.
func (s *Session) ProgressSummary() string {
	if s == nil || len(s.Jobs) == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d complete", len(s.Jobs)-len(s.ResumableURLs()), len(s.Jobs))
}

// RunnerAlive reports whether the process that owns this session is running.
// A session that has not recorded a PID yet counts as neither alive nor stopped.
func (s *Session) RunnerAlive() bool {
	return s != nil && s.PID > 0 && processAlive(s.PID)
}

// RunnerStopped reports whether the process that owned this session is gone.
func (s *Session) RunnerStopped() bool {
	return s != nil && s.PID > 0 && !processAlive(s.PID)
}

// ActiveSession returns the session a live runner is working on, or nil if the
// last session already finished or its runner died.
func ActiveSession(rootDir string) *Session {
	s, err := LoadSession(rootDir)
	if err != nil || s == nil || s.Complete || !s.RunnerAlive() {
		return nil
	}
	return s
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
	case EventAllDone:
		s.Complete = true
		s.Phase = SessionPhaseDone
		for _, j := range s.Jobs {
			if j.Status == StatusCancelled {
				s.Phase = SessionPhaseCancelled
				break
			}
		}
	}
}
