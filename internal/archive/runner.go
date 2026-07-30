package archive

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

const pollInterval = 200 * time.Millisecond

// StartSessionRunner validates the session, persists it, and launches a
// detached "archive session-run" subprocess to crawl it.
func StartSessionRunner(rootDir string, session *Session) error {
	if session == nil || len(session.Jobs) == 0 {
		return errors.New("no URLs to archive")
	}
	for _, j := range session.Jobs {
		if !ValidURL(j.URL) {
			return fmt.Errorf("URL must start with http:// or https://: %s", j.URL)
		}
	}
	if err := DockerAvailable(); err != nil {
		return err
	}
	if ActiveSession(rootDir) != nil {
		return errors.New("a crawl is already running; reattach with archive, or stop it with archive quit")
	}

	if err := os.MkdirAll(filepath.Join(rootDir, "crawls"), 0o777); err != nil {
		return err
	}
	if err := os.WriteFile(sessionLogPath(rootDir), nil, 0o644); err != nil {
		return err
	}
	session.PID = 0
	if err := SaveSession(rootDir, session); err != nil {
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, "session-run")
	cmd.Dir = rootDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()

	// RunSession records the same PID from inside the child, so whichever write
	// lands last, the session ends up pointing at the runner process.
	session.PID = cmd.Process.Pid
	return SaveSession(rootDir, session)
}

// RunSession crawls the persisted session in the foreground. It backs the
// "archive session-run" subcommand, which StartSessionRunner spawns.
func RunSession(rootDir string) error {
	cfg, err := LoadConfig(rootDir)
	if err != nil {
		return err
	}
	session, err := LoadSession(rootDir)
	if err != nil {
		return err
	}
	if session == nil || len(session.Jobs) == 0 {
		return errors.New("no session to run")
	}
	session.PID = os.Getpid()
	if err := SaveSession(rootDir, session); err != nil {
		return err
	}

	orch := NewOrchestrator(cfg, session)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go watchControl(ctx, rootDir, orch, cancel)
	go orch.Run(ctx)

	for ev := range orch.Events {
		session.ApplyEvent(ev)
		if ev.Type == EventLog {
			_ = AppendSessionLog(rootDir, ev.Line)
		}
		_ = SaveSession(rootDir, session)
	}

	// The runner is exiting, so nothing can advance this session any further.
	session.PID = 0
	session.Complete = true
	return SaveSession(rootDir, session)
}

// watchControl relays skip/cancel commands sent by other archive processes.
func watchControl(ctx context.Context, rootDir string, orch *Orchestrator, cancel context.CancelFunc) {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			switch cmd, _ := ConsumeControl(rootDir); cmd {
			case "skip":
				orch.SkipCurrent()
			case "cancel":
				orch.CancelAll()
				cancel()
				return
			}
		}
	}
}

// RunHeadless starts a crawl session and streams its log to w until it finishes.
func RunHeadless(rootDir string, urls []string, w io.Writer) error {
	if err := StartSessionRunner(rootDir, NewSession(urls)); err != nil {
		return err
	}

	offset := 0
	for {
		time.Sleep(pollInterval)

		if logText, err := ReadSessionLog(rootDir); err == nil && len(logText) > offset {
			if _, err := io.WriteString(w, logText[offset:]); err != nil {
				return err
			}
			offset = len(logText)
		}

		session, err := LoadSession(rootDir)
		switch {
		case err != nil:
			return err
		case session == nil:
			return errors.New("session file disappeared")
		case session.Phase == SessionPhaseError:
			return errors.New(session.Error)
		case session.Complete:
			return nil
		case session.RunnerStopped():
			return errors.New("crawl runner stopped unexpectedly")
		}
	}
}
