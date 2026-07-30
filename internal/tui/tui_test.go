package tui

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/TaylorJadin/site-archiving-toolkit/internal/archive"
)

// appendRaw writes to the session log without a trailing newline, imitating a
// line the runner has not finished flushing.
func appendRaw(rootDir, text string) error {
	f, err := os.OpenFile(filepath.Join(rootDir, "crawls", ".last-session.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(text)
	return err
}

func newTestModel(t *testing.T) Model {
	t.Helper()
	return New(&archive.Config{RootDir: t.TempDir()}, nil)
}

// New must apply everything startSession does to the model it returns; an
// earlier version threw those mutations away, so a crawl started from the
// command line left the user staring at the input screen.
func TestNewWithURLsReportsStartFailure(t *testing.T) {
	m := New(&archive.Config{RootDir: t.TempDir()}, []string{"ftp://not-a-web-url"})
	if m.errMsg == "" {
		t.Fatalf("expected a start error on the model, got phase=%v", m.phase)
	}
	if !strings.Contains(m.errMsg, "http://") {
		t.Fatalf("unexpected error message %q", m.errMsg)
	}
}

// Exactly one poll may be in flight at a time. Previously both the poll and the
// spinner tick queued a new poll, so pending polls grew without bound.
func TestPollLoopDoesNotFanOut(t *testing.T) {
	m := newTestModel(t)
	m.phase = phaseRunning

	var model tea.Model = m
	queue := []tea.Msg{sessionPollMsg{session: &archive.Session{}}, m.spinner.Tick()}
	for round := range 5 {
		var next []tea.Msg
		for _, msg := range queue {
			var cmd tea.Cmd
			model, cmd = model.Update(msg)
			next = append(next, evaluate(cmd)...)
		}
		polls := 0
		for _, msg := range next {
			if _, ok := msg.(sessionPollMsg); ok {
				polls++
			}
		}
		if polls != 1 {
			t.Fatalf("round %d: %d polls in flight, want 1", round, polls)
		}
		queue = next
	}
}

// evaluate runs a command, flattening batches, and returns the messages produced.
func evaluate(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, evaluate(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

func TestApplyPollFinishesRun(t *testing.T) {
	cases := []struct {
		name      string
		session   *archive.Session
		wantPhase phase
	}{
		{
			name:      "complete",
			session:   &archive.Session{Complete: true, Phase: archive.SessionPhaseDone, Jobs: []archive.Job{{URL: "https://a.com", Status: archive.StatusCompleted}}},
			wantPhase: phaseDone,
		},
		{
			// Quitting cancels jobs; the run is still over and must not hang.
			name:      "cancelled",
			session:   &archive.Session{Complete: true, Phase: archive.SessionPhaseCancelled, Jobs: []archive.Job{{URL: "https://a.com", Status: archive.StatusCancelled}}},
			wantPhase: phaseDone,
		},
		{
			name:      "error",
			session:   &archive.Session{Complete: true, Phase: archive.SessionPhaseError, Error: "docker build failed"},
			wantPhase: phaseError,
		},
		{
			name:      "runner died",
			session:   &archive.Session{PID: 1 << 30, Phase: archive.SessionPhaseCrawling},
			wantPhase: phaseError,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t)
			m.phase = phaseRunning

			next, cmd := m.Update(sessionPollMsg{session: tc.session})
			got := next.(Model)
			if got.phase != tc.wantPhase {
				t.Fatalf("phase = %v, want %v", got.phase, tc.wantPhase)
			}
			if cmd != nil {
				t.Fatal("expected polling to stop once the run is over")
			}
		})
	}
}

func TestApplyPollKeepsPollingWhileRunning(t *testing.T) {
	m := newTestModel(t)
	m.phase = phaseRunning

	next, cmd := m.Update(sessionPollMsg{
		session:   &archive.Session{Phase: archive.SessionPhaseCrawling, Jobs: []archive.Job{{URL: "https://a.com"}}},
		logChunk:  "one\ntwo\n",
		logOffset: 8,
	})
	got := next.(Model)
	if cmd == nil {
		t.Fatal("expected polling to continue")
	}
	if !reflect.DeepEqual(got.logs, []string{"one", "two"}) {
		t.Fatalf("logs = %v", got.logs)
	}
	if got.logOffset != 8 {
		t.Fatalf("logOffset = %d, want 8", got.logOffset)
	}
}

// A log line still being written must not be split across two updates.
func TestPollSessionOnlyConsumesWholeLines(t *testing.T) {
	dir := t.TempDir()
	if err := archive.SaveSession(dir, archive.NewSession([]string{"https://a.com"})); err != nil {
		t.Fatal(err)
	}
	if err := archive.AppendSessionLog(dir, "complete line"); err != nil {
		t.Fatal(err)
	}
	// Simulate a line that the runner has not finished writing.
	if err := appendRaw(dir, "partial"); err != nil {
		t.Fatal(err)
	}

	msg := pollSession(dir, 0)().(sessionPollMsg)
	if msg.logChunk != "complete line\n" {
		t.Fatalf("logChunk = %q", msg.logChunk)
	}
	if msg.logOffset != len("complete line\n") {
		t.Fatalf("logOffset = %d", msg.logOffset)
	}
}

func TestQuitTwiceExitsImmediately(t *testing.T) {
	m := newTestModel(t)
	m.phase = phaseRunning

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'q', Text: "q"})
	got := next.(Model)
	if !got.quitting || cmd != nil {
		t.Fatalf("first quit should request cancellation and wait, got quitting=%v cmd=%v", got.quitting, cmd != nil)
	}
	if _, cmd = got.Update(tea.KeyPressMsg{Code: 'q', Text: "q"}); cmd == nil {
		t.Fatal("second quit should exit the program")
	}
}

func TestDetachPrintsFarewell(t *testing.T) {
	m := newTestModel(t)
	m.phase = phaseRunning

	next, cmd := m.Update(tea.KeyPressMsg{Code: 'd', Text: "d"})
	if cmd == nil {
		t.Fatal("expected detach to quit the program")
	}
	if !strings.Contains(next.(Model).farewell, "background") {
		t.Fatalf("farewell = %q", next.(Model).farewell)
	}
}

// r resumes only when nothing has been typed, so URLs containing an r are not
// swallowed by the shortcut.
func TestResumeShortcutOnlyWhenInputIsEmpty(t *testing.T) {
	dir := t.TempDir()
	// The URL is deliberately invalid: the session still counts as resumable,
	// but a regression here cannot reach Docker and start a real crawl.
	session := archive.NewSession([]string{"ftp://a.com"})
	session.Complete = true
	if err := archive.SaveSession(dir, session); err != nil {
		t.Fatal(err)
	}

	m := New(&archive.Config{RootDir: dir}, nil)
	if !m.canResume {
		t.Fatal("expected the unfinished session to be resumable")
	}

	typed, _ := m.Update(tea.KeyPressMsg{Code: 'h', Text: "h"})
	typed, _ = typed.(Model).Update(tea.KeyPressMsg{Code: 'r', Text: "r"})
	got := typed.(Model)
	if got.phase != phaseInput {
		t.Fatalf("phase = %v, want the input screen", got.phase)
	}
	if got.textarea.Value() != "hr" {
		t.Fatalf("textarea = %q, want %q", got.textarea.Value(), "hr")
	}
}

func TestPasteMultilineURLs(t *testing.T) {
	m := newTestModel(t)
	next, _ := m.Update(tea.PasteMsg{Content: "https://a.com\nhttps://b.com\nhttps://c.com"})

	value := next.(Model).textarea.Value()
	if lines := strings.Split(strings.TrimRight(value, "\n"), "\n"); len(lines) != 3 {
		t.Fatalf("expected 3 URL lines after paste, got %d: %q", len(lines), value)
	}
}

func TestViewInputHasNoBoxBorder(t *testing.T) {
	m := newTestModel(t)
	m.width, m.height = 80, 24
	m.resize()

	view := m.viewInput()
	if !strings.Contains(view, "Enter URL(s) to crawl") {
		t.Fatalf("missing subtitle in view:\n%s", view)
	}
	for _, ch := range []string{"╭", "╮", "╰", "╯"} {
		if strings.Contains(view, ch) {
			t.Fatalf("view still has boxed border corner %q:\n%s", ch, view)
		}
	}
}

func TestViewDoesNotUseAltScreen(t *testing.T) {
	if newTestModel(t).View().AltScreen {
		t.Fatal("expected alt-screen disabled")
	}
}

func TestViewRunningKeyHints(t *testing.T) {
	m := newTestModel(t)
	m.phase = phaseRunning
	m.width, m.height = 80, 24
	m.resize()

	view := m.viewRunning()
	for _, want := range []string{"s skip url", "d detach", "q quit"} {
		if !strings.Contains(view, want) {
			t.Fatalf("missing %q in running view:\n%s", want, view)
		}
	}

	m.building = true
	if view := m.viewRunning(); strings.Contains(view, "s skip url") {
		t.Fatalf("skip should be hidden while the image builds:\n%s", view)
	}
}
