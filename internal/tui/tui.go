package tui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/TaylorJadin/site-archiving-toolkit/internal/archive"
)

type phase int

const (
	phaseInput phase = iota
	phaseRunning
	phaseDone
	phaseError
)

const (
	maxLogLines  = 500
	pollInterval = 200 * time.Millisecond
)

// sessionPollMsg carries the latest session state and any new log output.
type sessionPollMsg struct {
	session   *archive.Session
	logChunk  string
	logOffset int
}

// Model is the Bubble Tea TUI model.
type Model struct {
	cfg    *archive.Config
	phase  phase
	width  int
	height int

	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	currentIdx  int
	total       int
	currentURL  string
	statusLabel string
	building    bool
	quitting    bool
	logs        []string
	results     []string
	errMsg      string
	logOffset   int

	// farewell is printed by Run after the program exits.
	farewell string

	canResume  bool
	resumeInfo string
}

var (
	colFoam      = lipgloss.Color("#E8F1F0")
	colSea       = lipgloss.Color("#2A6F6B")
	colSeaBright = lipgloss.Color("#3D9B95")
	colSand      = lipgloss.Color("#C4A574")
	colMuted     = lipgloss.Color("#6B7C7A")
	colDanger    = lipgloss.Color("#B85C38")
	colOk        = lipgloss.Color("#3D8B5F")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colSea).
			MarginBottom(1)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colMuted)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colSea).
			Padding(0, 1)

	btnStyle = lipgloss.NewStyle().
			Foreground(colFoam).
			Background(colSea).
			Padding(0, 2).
			MarginRight(1)

	btnDangerStyle = lipgloss.NewStyle().
			Foreground(colFoam).
			Background(colDanger).
			Padding(0, 2).
			MarginRight(1)

	hintStyle = lipgloss.NewStyle().
			Foreground(colMuted)

	statusStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colSeaBright)

	progressStyle = lipgloss.NewStyle().
			Foreground(colSand).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(colDanger).
			Bold(true)

	okStyle = lipgloss.NewStyle().
		Foreground(colOk)

	ruleStyle = lipgloss.NewStyle().
			Foreground(colMuted)
)

// New creates the initial TUI model. When urls is non-empty the crawl starts
// immediately; otherwise the model either reattaches to a running crawl or
// shows the URL input.
func New(cfg *archive.Config, urls []string) Model {
	ta := textarea.New()
	ta.Placeholder = "https://example.com https://another-site.org"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(72)
	ta.SetHeight(8)
	ta.ShowLineNumbers = false
	ta.Prompt = "│ "
	ta.KeyMap.InsertNewline = key.NewBinding(
		key.WithKeys("shift+enter", "ctrl+j"),
		key.WithHelp("shift+enter", "insert newline"),
	)
	styles := textarea.DefaultLightStyles()
	styles.Focused.Text = lipgloss.NewStyle().Foreground(colSea)
	styles.Focused.Placeholder = lipgloss.NewStyle().Foreground(colMuted)
	styles.Focused.Prompt = lipgloss.NewStyle().Foreground(colSea)
	styles.Focused.CursorLine = lipgloss.NewStyle()
	styles.Blurred.Text = lipgloss.NewStyle().Foreground(colMuted)
	styles.Blurred.Placeholder = lipgloss.NewStyle().Foreground(colMuted)
	styles.Blurred.Prompt = lipgloss.NewStyle().Foreground(colMuted)
	styles.Cursor.Color = colSea
	ta.SetStyles(styles)

	m := Model{
		cfg:      cfg,
		phase:    phaseInput,
		textarea: ta,
		viewport: viewport.New(viewport.WithWidth(72), viewport.WithHeight(12)),
		spinner: spinner.New(
			spinner.WithSpinner(spinner.Dot),
			spinner.WithStyle(lipgloss.NewStyle().Foreground(colSeaBright)),
		),
		currentIdx: -1,
	}

	switch {
	case len(urls) > 0:
		m.startSession(urls)
	default:
		if session := archive.ActiveSession(cfg.RootDir); session != nil {
			m.attach(session)
		} else if session, _ := archive.LoadSession(cfg.RootDir); session.CanResume() {
			m.canResume = true
			m.resumeInfo = session.ProgressSummary()
		}
	}
	return m
}

// startSession spawns a crawl runner and attaches to it, reporting success.
func (m *Model) startSession(urls []string) bool {
	session := archive.NewSession(urls)
	if err := archive.StartSessionRunner(m.cfg.RootDir, session); err != nil {
		m.errMsg = err.Error()
		return false
	}
	m.attach(session)
	return true
}

func (m *Model) attach(session *archive.Session) {
	m.phase = phaseRunning
	m.errMsg = ""
	m.results = nil
	m.logs = nil
	m.logOffset = 0
	m.sync(session)
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.phase == phaseRunning {
		return m.watch()
	}
	return textarea.Blink
}

// watch drives the spinner and the session poll loop.
func (m Model) watch() tea.Cmd {
	return tea.Batch(m.spinner.Tick, pollSession(m.cfg.RootDir, m.logOffset))
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()
		return m, nil

	case tea.KeyPressMsg:
		switch m.phase {
		case phaseInput:
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "enter":
				return m.startArchive()
			case "r", "R":
				// Only a resume shortcut while the box is empty, so URLs
				// containing an r can still be typed.
				if m.canResume && strings.TrimSpace(m.textarea.Value()) == "" {
					return m.resumeLastSession()
				}
			}
		case phaseRunning:
			switch msg.String() {
			case "s", "S":
				if !m.building {
					_ = archive.SendControl(m.cfg.RootDir, "skip")
					m.statusLabel = "Skipping URL..."
				}
				return m, nil
			case "d", "D":
				m.farewell = "Detached — crawl continues in background. Run archive to reattach."
				return m, tea.Quit
			case "q", "Q", "ctrl+c", "esc":
				return m.cancelSession()
			}
		case phaseDone, phaseError:
			switch msg.String() {
			case "q", "esc", "ctrl+c", "enter":
				return m, tea.Quit
			case "r", "R":
				next := New(m.cfg, nil)
				next.width, next.height = m.width, m.height
				next.resize()
				return next, next.Init()
			}
		}

	case sessionPollMsg:
		return m.applyPoll(msg)

	case spinner.TickMsg:
		if m.phase != phaseRunning {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Anything else (text input, paste, scroll) goes to the focused component.
	var cmd tea.Cmd
	switch m.phase {
	case phaseInput:
		m.textarea, cmd = m.textarea.Update(msg)
	case phaseRunning:
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

func (m Model) startArchive() (Model, tea.Cmd) {
	urls := archive.ParseURLs(m.textarea.Value())
	if len(urls) == 0 {
		m.errMsg = "Enter at least one URL"
		return m, nil
	}
	return m.start(urls)
}

func (m Model) resumeLastSession() (Model, tea.Cmd) {
	session, err := archive.LoadSession(m.cfg.RootDir)
	if err != nil || !session.CanResume() {
		m.errMsg = "No session to resume"
		m.canResume = false
		return m, nil
	}
	return m.start(session.ResumableURLs())
}

func (m Model) start(urls []string) (Model, tea.Cmd) {
	if !m.startSession(urls) {
		return m, nil
	}
	if m.cfg.BackgroundModeDefault {
		m.farewell = fmt.Sprintf("Crawl started in background (%d URLs). Run archive to reattach.", m.total)
		return m, tea.Quit
	}
	return m, m.watch()
}

func (m Model) cancelSession() (Model, tea.Cmd) {
	if m.quitting {
		// Second press: stop waiting for the runner to wind down.
		return m, tea.Quit
	}
	_ = archive.SendControl(m.cfg.RootDir, "cancel")
	m.quitting = true
	m.statusLabel = "Quitting..."
	return m, nil
}

func (m Model) applyPoll(msg sessionPollMsg) (Model, tea.Cmd) {
	if m.phase != phaseRunning {
		return m, nil
	}
	m.appendLogs(msg.logChunk)
	m.logOffset = msg.logOffset

	session := msg.session
	if session == nil {
		// The session file is missing or unreadable; try again next tick.
		return m, pollSession(m.cfg.RootDir, m.logOffset)
	}
	m.sync(session)

	switch {
	case session.Phase == archive.SessionPhaseError:
		m.phase = phaseError
		m.errMsg = session.Error
	case session.Complete:
		m.phase = phaseDone
	case session.RunnerStopped():
		m.phase = phaseError
		m.errMsg = "crawl runner stopped unexpectedly"
	default:
		return m, pollSession(m.cfg.RootDir, m.logOffset)
	}

	m.rebuildResults(session)
	return m, nil
}

func (m *Model) sync(session *archive.Session) {
	m.total = len(session.Jobs)
	m.currentIdx = session.CurrentIndex
	m.building = session.Phase == archive.SessionPhaseBuilding
	if m.quitting && !session.Complete {
		m.statusLabel = "Quitting..."
	} else {
		m.statusLabel = phaseLabel(session.Phase)
	}
	if session.CurrentIndex >= 0 && session.CurrentIndex < len(session.Jobs) {
		m.currentURL = session.Jobs[session.CurrentIndex].URL
	}
}

func phaseLabel(p archive.SessionPhase) string {
	switch p {
	case archive.SessionPhaseBuilding:
		return "Building image..."
	case archive.SessionPhaseCrawling:
		return "Crawling"
	case archive.SessionPhaseCancelled:
		return "Cancelled"
	case archive.SessionPhaseError:
		return "Error"
	default:
		return "Complete"
	}
}

func (m *Model) rebuildResults(session *archive.Session) {
	m.results = nil
	for _, job := range session.Jobs {
		if job.Status == archive.StatusPending || job.Status == archive.StatusRunning {
			continue
		}
		m.results = append(m.results, fmt.Sprintf("%s  %s  (%s)", statusGlyph(job.Status), job.URL, job.Status))
	}
}

func (m *Model) appendLogs(chunk string) {
	if chunk == "" {
		return
	}
	for _, line := range strings.Split(strings.TrimSuffix(chunk, "\n"), "\n") {
		m.logs = append(m.logs, line)
	}
	if len(m.logs) > maxLogLines {
		m.logs = m.logs[len(m.logs)-maxLogLines:]
	}
	m.viewport.SetContent(strings.Join(m.logs, "\n"))
	m.viewport.GotoBottom()
}

// pollSession reloads the session and any log output written since offset.
// Only whole lines are consumed so a partially written line is never split.
func pollSession(rootDir string, offset int) tea.Cmd {
	return tea.Tick(pollInterval, func(time.Time) tea.Msg {
		session, _ := archive.LoadSession(rootDir)
		logText, _ := archive.ReadSessionLog(rootDir)

		msg := sessionPollMsg{session: session, logOffset: offset}
		if len(logText) > offset {
			chunk := logText[offset:]
			if end := strings.LastIndexByte(chunk, '\n'); end >= 0 {
				msg.logChunk = chunk[:end+1]
				msg.logOffset = offset + end + 1
			}
		}
		return msg
	})
}

func statusGlyph(s archive.CrawlStatus) string {
	switch s {
	case archive.StatusCompleted:
		return "✓"
	case archive.StatusSkipped:
		return "↷"
	case archive.StatusFailed:
		return "✗"
	case archive.StatusCancelled:
		return "■"
	default:
		return "·"
	}
}

func (m *Model) resize() {
	m.textarea.SetWidth(clamp(m.width-4, 40, 100))
	m.viewport.SetWidth(max(m.width, 40))
	m.viewport.SetHeight(max(m.height-12, 6))
}

func clamp(v, lo, hi int) int { return min(max(v, lo), hi) }

// View implements tea.Model.
func (m Model) View() tea.View {
	switch m.phase {
	case phaseRunning:
		return tea.NewView(m.viewRunning())
	case phaseDone:
		return tea.NewView(m.viewDone())
	case phaseError:
		return tea.NewView(m.viewError())
	default:
		return tea.NewView(m.viewInput())
	}
}

func header(b *strings.Builder) {
	b.WriteString(titleStyle.Render("Site Archiving Toolkit"))
	b.WriteString("\n")
}

func (m Model) viewInput() string {
	var b strings.Builder
	header(&b)
	b.WriteString(subtitleStyle.Render("Enter URL(s) to crawl (multiple URLs can be separated by a space or new line)"))
	b.WriteString("\n\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n\n")
	if m.canResume {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf("Last session incomplete (%s)", m.resumeInfo)))
		b.WriteString("\n")
		b.WriteString(btnStyle.Render("r resume"))
		b.WriteString("\n\n")
	}
	if m.errMsg != "" {
		b.WriteString(errorStyle.Render(m.errMsg))
		b.WriteString("\n\n")
	}
	b.WriteString(btnStyle.Render("enter start"))
	b.WriteString(hintStyle.Render("  shift+enter / ctrl+j new line · esc quit"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) viewRunning() string {
	var b strings.Builder
	header(&b)

	progress := ""
	if m.building {
		progress = progressStyle.Render("Building image…")
	} else if m.total > 0 && m.currentIdx >= 0 {
		progress = progressStyle.Render(fmt.Sprintf("Site %d/%d", m.currentIdx+1, m.total))
	}
	fmt.Fprintf(&b, "%s %s  %s\n", m.spinner.View(), progress, statusStyle.Render(m.statusLabel))
	if m.currentURL != "" && !m.building {
		b.WriteString(subtitleStyle.Render(m.currentURL))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if !m.building {
		b.WriteString(btnStyle.Render("s skip url"))
	}
	b.WriteString(btnStyle.Render("d detach"))
	b.WriteString(btnDangerStyle.Render("q quit"))
	if !m.building {
		b.WriteString(hintStyle.Render("  ↑↓ scroll logs"))
	}
	b.WriteString("\n")

	b.WriteString(ruleStyle.Render(strings.Repeat("─", max(m.viewport.Width(), 40))))
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder
	header(&b)
	b.WriteString(okStyle.Render("Archive run finished"))
	b.WriteString("\n\n")
	if len(m.results) > 0 {
		b.WriteString(boxStyle.Render(strings.Join(m.results, "\n")))
		b.WriteString("\n\n")
	}
	b.WriteString(hintStyle.Render("q quit · r new archive"))
	b.WriteString("\n")
	return b.String()
}

func (m Model) viewError() string {
	var b strings.Builder
	header(&b)
	b.WriteString(errorStyle.Render("Error"))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(m.errMsg))
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("q quit · r try again"))
	b.WriteString("\n")
	return b.String()
}

// Run starts the Bubble Tea program. When urls is non-empty the crawl starts
// right away, otherwise the TUI opens on the URL input (or reattaches to a
// crawl that is already running).
func Run(cfg *archive.Config, urls []string) error {
	m := New(cfg, urls)
	if len(urls) > 0 && m.phase != phaseRunning {
		return errors.New(m.errMsg)
	}

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return err
	}
	if fm, ok := final.(Model); ok && fm.farewell != "" {
		fmt.Println(fm.farewell)
	}
	return nil
}
