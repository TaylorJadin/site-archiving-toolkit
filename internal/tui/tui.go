package tui

import (
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
	"github.com/TaylorJadin/site-archiving-toolkit/internal/config"
)

type phase int

const (
	phaseInput phase = iota
	phaseRunning
	phaseDone
	phaseError
)

type sessionPollMsg struct {
	session  *archive.Session
	logChunk string
}

// Options configures TUI startup behavior.
type Options struct {
	URLs       []string
	Background bool
	Resume     bool
}

// Model is the Bubble Tea TUI model.
type Model struct {
	cfg    *config.Config
	opts   Options
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
	logs        []string
	results     []string
	errMsg      string

	attached   bool
	detaching  bool
	logOffset  int
	canResume  bool
	resumeInfo string
}

const maxLogLines = 500

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

// New creates the initial TUI model.
func New(cfg *config.Config, opts Options) Model {
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

	sp := spinner.New(
		spinner.WithSpinner(spinner.Dot),
		spinner.WithStyle(lipgloss.NewStyle().Foreground(colSeaBright)),
	)

	vp := viewport.New(viewport.WithWidth(72), viewport.WithHeight(12))
	vp.SetContent("")

	m := Model{
		cfg:         cfg,
		opts:        opts,
		phase:       phaseInput,
		textarea:    ta,
		viewport:    vp,
		spinner:     sp,
		statusLabel: "Ready",
		currentIdx:  -1,
	}

	m.initStartupState()
	return m
}

func (m *Model) initStartupState() {
	if active, session := archive.IsRunnerActive(m.cfg.RootDir); active && session != nil {
		m.attachToSession(session)
		return
	}

	if m.opts.Resume {
		if session, _ := archive.LoadSession(m.cfg.RootDir); session != nil && session.CanResume() {
			m.startSession(session.ResumableURLs(), m.opts.Background)
			return
		}
	}

	if len(m.opts.URLs) > 0 {
		bg := m.opts.Background || m.cfg.BackgroundModeDefault
		m.startSession(m.opts.URLs, bg)
		return
	}

	if session, _ := archive.LoadSession(m.cfg.RootDir); session != nil && session.CanResume() {
		m.canResume = true
		m.resumeInfo = session.ProgressSummary()
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	if m.phase == phaseRunning {
		return tea.Batch(textarea.Blink, m.spinner.Tick, pollSession(m.cfg.RootDir, 0))
	}
	return textarea.Blink
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
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
				if m.canResume {
					return m.resumeLastSession()
				}
			}
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd

		case phaseRunning:
			switch msg.String() {
			case "s", "S":
				if !m.building {
					_ = archive.SendControl(m.cfg.RootDir, "skip")
					m.statusLabel = "Skipping URL..."
				}
				return m, nil
			case "d", "D":
				return m.detachSession()
			case "q", "Q", "ctrl+c", "esc":
				return m.cancelSession()
			}
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd

		case phaseDone, phaseError:
			switch msg.String() {
			case "q", "esc", "ctrl+c", "enter":
				return m, tea.Quit
			case "r":
				w, h := m.width, m.height
				m = New(m.cfg, Options{})
				m.width, m.height = w, h
				m.resize()
				return m, textarea.Blink
			}
		}

	case tea.PasteMsg:
		if m.phase == phaseInput {
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd
		}

	case spinner.TickMsg:
		if m.phase == phaseRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, tea.Batch(cmd, pollSession(m.cfg.RootDir, m.logOffset))
		}

	case sessionPollMsg:
		return m.applySessionPoll(msg)
	}

	if m.phase == phaseInput {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		return m, cmd
	}
	if m.phase == phaseRunning {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) startArchive() (Model, tea.Cmd) {
	urls := archive.ParseURLs(m.textarea.Value())
	if len(urls) == 0 {
		m.errMsg = "Enter at least one URL"
		return m, nil
	}
	bg := m.cfg.BackgroundModeDefault
	return m.startSession(urls, bg)
}

func (m Model) resumeLastSession() (Model, tea.Cmd) {
	session, err := archive.LoadSession(m.cfg.RootDir)
	if err != nil || session == nil || !session.CanResume() {
		m.errMsg = "No session to resume"
		return m, nil
	}
	return m.startSession(session.ResumableURLs(), m.cfg.BackgroundModeDefault)
}

func (m Model) startSession(urls []string, background bool) (Model, tea.Cmd) {
	session := archive.NewSession(urls)
	if _, err := archive.StartSessionRunner(m.cfg.RootDir, session); err != nil {
		m.errMsg = err.Error()
		return m, nil
	}

	if background {
		fmt.Printf("Crawl started in background (%d URLs). Run archive to reattach.\n", len(urls))
		return m, tea.Quit
	}

	m.attachToSession(session)
	return m, tea.Batch(m.spinner.Tick, pollSession(m.cfg.RootDir, 0))
}

func (m *Model) attachToSession(session *archive.Session) {
	m.attached = true
	m.phase = phaseRunning
	m.building = session.Phase == archive.SessionPhaseBuilding
	m.total = len(session.Jobs)
	m.currentIdx = session.CurrentIndex
	m.errMsg = ""
	m.results = nil
	m.logs = nil
	m.logOffset = 0

	if logText, err := archive.ReadSessionLog(m.cfg.RootDir); err == nil && logText != "" {
		m.syncLogs(logText)
	}
	m.syncFromSession(session)
}

func (m Model) applySessionPoll(msg sessionPollMsg) (Model, tea.Cmd) {
	if m.phase != phaseRunning {
		return m, nil
	}
	if msg.logChunk != "" {
		m.syncLogs(msg.logChunk)
	}
	if msg.session != nil {
		m.syncFromSession(msg.session)
		if msg.session.Complete {
			if msg.session.Phase == archive.SessionPhaseError {
				m.phase = phaseError
				m.errMsg = msg.session.Error
			} else {
				m.phase = phaseDone
				m.statusLabel = "Complete"
			}
			m.rebuildResults(msg.session)
			return m, nil
		}
	}
	return m, tea.Batch(m.spinner.Tick, pollSession(m.cfg.RootDir, m.logOffset))
}

func (m *Model) syncFromSession(session *archive.Session) {
	if session == nil {
		return
	}
	m.total = len(session.Jobs)
	m.currentIdx = session.CurrentIndex
	m.building = session.Phase == archive.SessionPhaseBuilding
	switch session.Phase {
	case archive.SessionPhaseBuilding:
		m.statusLabel = "Building image..."
	case archive.SessionPhaseCrawling:
		m.statusLabel = "Crawling"
	case archive.SessionPhaseDone:
		m.statusLabel = "Complete"
	case archive.SessionPhaseCancelled:
		m.statusLabel = "Cancelled"
	case archive.SessionPhaseError:
		m.statusLabel = "Error"
	}
	if session.CurrentIndex >= 0 && session.CurrentIndex < len(session.Jobs) {
		m.currentURL = session.Jobs[session.CurrentIndex].URL
	}
}

func (m *Model) rebuildResults(session *archive.Session) {
	if session == nil {
		return
	}
	m.results = nil
	for _, job := range session.Jobs {
		if job.Status == archive.StatusPending || job.Status == archive.StatusRunning {
			continue
		}
		m.results = append(m.results, fmt.Sprintf("%s  %s  (%s)", statusGlyph(job.Status), job.URL, job.Status))
	}
}

func (m *Model) syncLogs(full string) {
	if full == "" {
		return
	}
	if len(full) <= m.logOffset {
		return
	}
	chunk := full[m.logOffset:]
	m.logOffset = len(full)
	for _, line := range strings.Split(chunk, "\n") {
		if line == "" {
			continue
		}
		m.appendLog(line)
	}
}

func (m Model) cancelSession() (Model, tea.Cmd) {
	if m.detaching {
		return m, tea.Quit
	}
	_ = archive.SendControl(m.cfg.RootDir, "cancel")
	m.statusLabel = "Quitting..."
	return m, pollSession(m.cfg.RootDir, m.logOffset)
}

func (m Model) detachSession() (Model, tea.Cmd) {
	if m.phase != phaseRunning {
		return m, tea.Quit
	}
	m.detaching = true
	if session, _ := archive.LoadSession(m.cfg.RootDir); session != nil {
		session.Detached = true
		_ = archive.SaveSession(m.cfg.RootDir, session)
	}
	fmt.Println("Detached — crawl continues in background. Run archive to reattach.")
	return m, tea.Quit
}

func pollSession(rootDir string, offset int) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(200 * time.Millisecond)
		session, _ := archive.LoadSession(rootDir)
		logText, _ := archive.ReadSessionLog(rootDir)
		chunk := ""
		if len(logText) > offset {
			chunk = logText[offset:]
		}
		return sessionPollMsg{session: session, logChunk: chunk}
	}
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

func (m *Model) appendLog(line string) {
	m.logs = append(m.logs, line)
	if len(m.logs) > maxLogLines {
		m.logs = m.logs[len(m.logs)-maxLogLines:]
	}
	m.viewport.SetContent(strings.Join(m.logs, "\n"))
	m.viewport.GotoBottom()
}

func (m *Model) resize() {
	inputW := m.width - 4
	if inputW < 40 {
		inputW = 40
	}
	if inputW > 100 {
		inputW = 100
	}
	m.textarea.SetWidth(inputW)

	logW := m.width
	if logW < 40 {
		logW = 40
	}
	logH := m.height - 12
	if logH < 6 {
		logH = 6
	}
	m.viewport.SetWidth(logW)
	m.viewport.SetHeight(logH)
}

func (m Model) content() string {
	switch m.phase {
	case phaseInput:
		return m.viewInput()
	case phaseRunning:
		return m.viewRunning()
	case phaseDone:
		return m.viewDone()
	case phaseError:
		return m.viewError()
	default:
		return ""
	}
}

// View implements tea.Model.
func (m Model) View() tea.View {
	return tea.NewView(m.content())
}

func (m Model) viewInput() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Site Archiving Toolkit"))
	b.WriteString("\n")
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
	b.WriteString(titleStyle.Render("Site Archiving Toolkit"))
	b.WriteString("\n")

	spin := m.spinner.View()
	progress := ""
	if m.building {
		progress = progressStyle.Render("Building image…")
	} else if m.total > 0 && m.currentIdx >= 0 {
		progress = progressStyle.Render(fmt.Sprintf("Site %d/%d", m.currentIdx+1, m.total))
	}
	b.WriteString(fmt.Sprintf("%s %s  %s\n", spin, progress, statusStyle.Render(m.statusLabel)))
	if m.currentURL != "" && !m.building {
		b.WriteString(subtitleStyle.Render(m.currentURL))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if !m.building {
		b.WriteString(btnStyle.Render("s skip url"))
		b.WriteString(btnStyle.Render("d detach"))
		b.WriteString(btnDangerStyle.Render("q quit"))
		b.WriteString(hintStyle.Render("  ↑↓ scroll logs"))
		b.WriteString("\n")
	} else {
		b.WriteString(btnStyle.Render("d detach"))
		b.WriteString(btnDangerStyle.Render("q quit"))
		b.WriteString("\n")
	}

	ruleW := m.viewport.Width()
	if ruleW < 1 {
		ruleW = 40
	}
	b.WriteString(ruleStyle.Render(strings.Repeat("─", ruleW)))
	b.WriteString("\n")
	b.WriteString(m.viewport.View())
	b.WriteString("\n")
	return b.String()
}

func (m Model) viewDone() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Site Archiving Toolkit"))
	b.WriteString("\n")
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
	b.WriteString(titleStyle.Render("Site Archiving Toolkit"))
	b.WriteString("\n")
	b.WriteString(errorStyle.Render("Error"))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(m.errMsg))
	b.WriteString("\n\n")
	b.WriteString(hintStyle.Render("q quit · r try again"))
	b.WriteString("\n")
	return b.String()
}

// Run starts the Bubble Tea program.
func Run(cfg *config.Config, opts Options) error {
	p := tea.NewProgram(New(cfg, opts))
	_, err := p.Run()
	return err
}
