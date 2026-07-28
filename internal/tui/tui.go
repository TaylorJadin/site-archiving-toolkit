package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

type crawlEventMsg archive.Event
type crawlDoneMsg struct{}

// Model is the Bubble Tea TUI model.
type Model struct {
	cfg    *config.Config
	phase  phase
	width  int
	height int

	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	orch   *archive.Orchestrator
	cancel context.CancelFunc

	currentIdx  int
	total       int
	currentURL  string
	statusLabel string
	building    bool
	logs        []string
	results     []string
	errMsg      string
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

	logBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colMuted).
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
)

// New creates the initial TUI model.
func New(cfg *config.Config) Model {
	ta := textarea.New()
	ta.Placeholder = "https://example.com\nhttps://another-site.org"
	ta.Focus()
	ta.CharLimit = 0
	ta.SetWidth(72)
	ta.SetHeight(8)
	ta.ShowLineNumbers = false
	ta.Prompt = "│ "

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(colSeaBright)

	vp := viewport.New(72, 12)
	vp.SetContent("")

	return Model{
		cfg:         cfg,
		phase:       phaseInput,
		textarea:    ta,
		viewport:    vp,
		spinner:     sp,
		statusLabel: "Ready",
		currentIdx:  -1,
	}
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
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

	case tea.KeyMsg:
		switch m.phase {
		case phaseInput:
			switch msg.String() {
			case "ctrl+c", "esc":
				return m, tea.Quit
			case "ctrl+s":
				return m.startArchive()
			}
			var cmd tea.Cmd
			m.textarea, cmd = m.textarea.Update(msg)
			return m, cmd

		case phaseRunning:
			switch msg.String() {
			case "s", "S":
				if m.orch != nil && !m.building {
					m.orch.SkipCurrent()
					m.statusLabel = "Skipping..."
				}
				return m, nil
			case "c", "C", "ctrl+c":
				if m.orch != nil {
					m.orch.CancelAll()
					if m.cancel != nil {
						m.cancel()
					}
					m.statusLabel = "Cancelling..."
				}
				return m, nil
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
				m = New(m.cfg)
				m.width, m.height = w, h
				m.resize()
				return m, textarea.Blink
			}
		}

	case spinner.TickMsg:
		if m.phase == phaseRunning {
			var cmd tea.Cmd
			m.spinner, cmd = m.spinner.Update(msg)
			return m, cmd
		}

	case crawlEventMsg:
		return m.handleEvent(archive.Event(msg))

	case crawlDoneMsg:
		if m.phase == phaseRunning {
			m.phase = phaseDone
			m.statusLabel = "All done"
		}
		return m, nil
	}

	return m, nil
}

func (m Model) startArchive() (Model, tea.Cmd) {
	raw := m.textarea.Value()
	lines := strings.Split(raw, "\n")
	urls := make([]string, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		urls = append(urls, line)
	}
	if len(urls) == 0 {
		m.errMsg = "Enter at least one URL"
		return m, nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.orch = archive.NewOrchestrator(m.cfg, urls)
	m.phase = phaseRunning
	m.building = true
	m.total = len(m.orch.Jobs)
	m.currentIdx = 0
	m.statusLabel = "Building image..."
	m.logs = nil
	m.results = nil
	m.errMsg = ""
	m.viewport.SetContent("")

	orch := m.orch
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			go orch.Run(ctx)
			return nil
		},
		listenEvents(orch),
	)
}

func listenEvents(orch *archive.Orchestrator) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-orch.Events
		if !ok {
			return crawlDoneMsg{}
		}
		return crawlEventMsg(ev)
	}
}

func (m Model) handleEvent(ev archive.Event) (Model, tea.Cmd) {
	switch ev.Type {
	case archive.EventLog:
		m.appendLog(ev.Line)
	case archive.EventBuildStarted:
		m.building = true
		m.statusLabel = ev.Message
	case archive.EventBuildFinished:
		m.building = false
		m.statusLabel = ev.Message
	case archive.EventJobStarted:
		m.building = false
		m.currentIdx = ev.Index
		m.total = ev.Total
		m.currentURL = ev.URL
		m.statusLabel = "Crawling"
		m.logs = nil
		m.appendLog(fmt.Sprintf("── Crawl %d/%d: %s ──", ev.Index+1, ev.Total, ev.URL))
	case archive.EventJobFinished:
		m.results = append(m.results, fmt.Sprintf("%s  %s  (%s)", statusGlyph(ev.Status), ev.URL, ev.Status))
		m.statusLabel = string(ev.Status)
	case archive.EventError:
		m.phase = phaseError
		m.errMsg = ev.Message
		m.appendLog("ERROR: " + ev.Message)
		return m, nil
	case archive.EventAllDone:
		m.phase = phaseDone
		m.statusLabel = "Complete"
		return m, nil
	}

	cmd := listenEvents(m.orch)
	if m.phase == phaseRunning {
		return m, tea.Batch(cmd, m.spinner.Tick)
	}
	return m, cmd
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
	w := m.width - 4
	if w < 40 {
		w = 40
	}
	if w > 100 {
		w = 100
	}
	m.textarea.SetWidth(w)

	logH := m.height - 14
	if logH < 6 {
		logH = 6
	}
	m.viewport.Width = w
	m.viewport.Height = logH
}

// View implements tea.Model.
func (m Model) View() string {
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

func (m Model) viewInput() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Site Archiving Toolkit"))
	b.WriteString("\n")
	b.WriteString(subtitleStyle.Render("Webrecorder archives · enter one URL per line"))
	b.WriteString("\n\n")
	b.WriteString(boxStyle.Render(m.textarea.View()))
	b.WriteString("\n\n")
	if m.errMsg != "" {
		b.WriteString(errorStyle.Render(m.errMsg))
		b.WriteString("\n\n")
	}
	b.WriteString(btnStyle.Render("ctrl+s start"))
	b.WriteString(hintStyle.Render("  esc quit"))
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
	} else if m.total > 0 {
		progress = progressStyle.Render(fmt.Sprintf("Site %d/%d", m.currentIdx+1, m.total))
	}
	b.WriteString(fmt.Sprintf("%s %s  %s\n", spin, progress, statusStyle.Render(m.statusLabel)))
	if m.currentURL != "" && !m.building {
		b.WriteString(subtitleStyle.Render(m.currentURL))
		b.WriteString("\n")
	}
	b.WriteString("\n")

	if !m.building {
		b.WriteString(btnStyle.Render("s skip"))
		b.WriteString(btnDangerStyle.Render("c cancel"))
		b.WriteString(hintStyle.Render("  ↑↓ scroll logs"))
		b.WriteString("\n\n")
	} else {
		b.WriteString(btnDangerStyle.Render("c cancel"))
		b.WriteString("\n\n")
	}

	header := lipgloss.NewStyle().Foreground(colMuted).Render("Crawl log")
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(logBoxStyle.Width(m.viewport.Width + 2).Render(m.viewport.View()))
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
func Run(cfg *config.Config) error {
	p := tea.NewProgram(New(cfg), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
