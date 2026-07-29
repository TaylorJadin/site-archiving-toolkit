package archive

import (
	"fmt"
	"io"
	"os"
	"time"
)

// WaitSession blocks until the current session finishes, streaming logs to w.
func WaitSession(rootDir string, w io.Writer) error {
	lastLog := 0
	for {
		session, err := LoadSession(rootDir)
		if err != nil {
			return err
		}
		if session == nil {
			return fmt.Errorf("session disappeared")
		}
		logText, err := ReadSessionLog(rootDir)
		if err != nil {
			return err
		}
		if len(logText) > lastLog {
			if _, err := io.WriteString(w, logText[lastLog:]); err != nil {
				return err
			}
			lastLog = len(logText)
		}
		if session.Complete {
			if session.Phase == SessionPhaseError {
				return fmt.Errorf("%s", session.Error)
			}
			return nil
		}
		active, _ := IsRunnerActive(rootDir)
		if !active && !session.Complete {
			return fmt.Errorf("session stopped unexpectedly")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// RunHeadless starts a crawl session and waits for completion without a TUI.
func RunHeadless(rootDir string, urls []string) error {
	if len(urls) == 0 {
		return fmt.Errorf("no URLs provided")
	}
	session := NewSession(urls)
	if _, err := StartSessionRunner(rootDir, session); err != nil {
		return err
	}
	return WaitSession(rootDir, os.Stdout)
}
