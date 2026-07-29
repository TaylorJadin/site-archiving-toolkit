package archive

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/TaylorJadin/site-archiving-toolkit/internal/config"
)

// StartSessionRunner launches a detached archive session-run subprocess.
func StartSessionRunner(rootDir string, session *Session) (int, error) {
	if session == nil || len(session.Jobs) == 0 {
		return 0, fmt.Errorf("empty session")
	}
	if err := os.MkdirAll(filepath.Join(rootDir, "crawls"), 0o777); err != nil {
		return 0, err
	}
	_, logPath, _ := SessionPaths(rootDir)
	if err := os.WriteFile(logPath, nil, 0o644); err != nil && !os.IsNotExist(err) {
		return 0, err
	}
	session.PID = 0
	session.Detached = false
	if err := SaveSession(rootDir, session); err != nil {
		return 0, err
	}

	exe, err := os.Executable()
	if err != nil {
		return 0, err
	}
	cmd := exec.Command(exe, "session-run")
	cmd.Dir = rootDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.Stdin = nil
	if err := cmd.Start(); err != nil {
		return 0, err
	}
	session.PID = cmd.Process.Pid
	if err := SaveSession(rootDir, session); err != nil {
		return 0, err
	}
	go func() { _ = cmd.Wait() }()
	return cmd.Process.Pid, nil
}

// RunSession executes the crawl session in the foreground (session-run command).
func RunSession(rootDir string) error {
	cfg, err := config.Load(rootDir)
	if err != nil {
		return err
	}
	session, err := LoadSession(rootDir)
	if err != nil {
		return err
	}
	if session == nil || len(session.Jobs) == 0 {
		return fmt.Errorf("no session to run")
	}

	orch := NewOrchestratorFromSession(cfg, session)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go watchSessionControl(ctx, rootDir, orch, cancel)

	eventsDone := make(chan struct{})
	go func() {
		defer close(eventsDone)
		for ev := range orch.Events {
			session.ApplyEvent(ev)
			if ev.Type == EventLog {
				_ = AppendSessionLog(rootDir, ev.Line)
			}
			_ = SaveSession(rootDir, session)
		}
	}()

	orch.Run(ctx)
	cancel()
	<-eventsDone

	session.PID = 0
	if !session.Complete {
		session.Resumable = len(session.ResumableURLs()) > 0
	}
	_ = SaveSession(rootDir, session)
	return nil
}

func watchSessionControl(ctx context.Context, rootDir string, orch *Orchestrator, cancel context.CancelFunc) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cmd, err := ConsumeControl(rootDir)
			if err != nil || cmd == "" {
				continue
			}
			switch cmd {
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
