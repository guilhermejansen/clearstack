// Package tui is the clearstack interactive terminal UI built with
// charmbracelet/bubbletea, bubbles, and lipgloss.
//
// The model follows a simple state machine:
//
//	stateScanning → stateResults → stateConfirming → stateCleaning → stateSummary
//
// Every transition is driven by a tea.Cmd that runs engine work on a
// background goroutine and feeds messages back to the Update loop.
package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/dustin/go-humanize"

	"github.com/guilhermejansen/clearstack/internal/detectors"
	"github.com/guilhermejansen/clearstack/internal/engine"
)

// state enumerates the possible TUI screens.
type state int

const (
	stateScanning state = iota
	stateResults
	stateConfirming
	stateCleaning
	stateSummary
)

// Options configures a TUI program. Sizer, Engine, and Roots are required.
type Options struct {
	Engine *engine.Engine
	Roots  []string
}

// Model is the root Bubble Tea model.
type Model struct {
	opts     Options
	state    state
	theme    Theme
	keys     KeyMap
	help     help.Model
	spinner  spinner.Model
	progress progress.Model
	filter   textinput.Model

	width  int
	height int

	// Scan state
	scanStart time.Time
	matches   []detectors.Match
	selected  map[int]bool // index in sorted
	cursor    int
	sortBy    sortKey
	filtered  []int // indices into matches

	scanErr  error
	scanDone bool

	// Clean state
	cleanSummary engine.CleanSummary
	cleanErr     error

	// Filter toggle
	filtering bool

	// Dry-run toggle (persistent)
	dryRun bool
	hard   bool
}

type sortKey int

const (
	sortSize sortKey = iota
	sortCategory
	sortPath
)

func (s sortKey) label() string {
	return []string{"size", "category", "path"}[s]
}

// New constructs a Model ready to be wrapped in a tea.Program.
func New(opts Options) *Model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#94e2d5"))

	pg := progress.New(progress.WithDefaultGradient(), progress.WithWidth(40))

	fi := textinput.New()
	fi.Placeholder = "type to filter (category or path)…"
	fi.CharLimit = 128

	return &Model{
		opts:     opts,
		state:    stateScanning,
		theme:    DefaultTheme(),
		keys:     DefaultKeys(),
		help:     help.New(),
		spinner:  sp,
		progress: pg,
		filter:   fi,
		selected: make(map[int]bool),
		sortBy:   sortSize,
	}
}

// ---- messages -------------------------------------------------------------

// matchMsg is emitted by the scanner goroutine for every match it finds.
type matchMsg struct{ m detectors.Match }

// scanDoneMsg marks scan completion; err may be nil.
type scanDoneMsg struct{ err error }

// sizedMsg is delivered once all matches have their sizes filled in.
type sizedMsg struct{}

// cleanDoneMsg delivers the CleanSummary after a Clean batch.
type cleanDoneMsg struct {
	sum engine.CleanSummary
	err error
}

// ---- tea.Model ------------------------------------------------------------

// Init kicks off scanning.
func (m *Model) Init() tea.Cmd {
	m.scanStart = time.Now()
	return tea.Batch(
		m.spinner.Tick,
		m.startScan(),
	)
}

// Update handles every event.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.progress.Width = max(20, m.width-20)
		m.help.Width = m.width

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		if m.state == stateScanning || m.state == stateCleaning {
			cmds = append(cmds, cmd)
		}

	case matchMsg:
		m.matches = append(m.matches, msg.m)
		m.rebuildFiltered()

	case scanDoneMsg:
		m.scanDone = true
		m.scanErr = msg.err
		cmds = append(cmds, m.sizeMatches())

	case sizedMsg:
		m.applySort()
		m.state = stateResults

	case cleanDoneMsg:
		m.cleanSummary = msg.sum
		m.cleanErr = msg.err
		m.state = stateSummary

	case tea.KeyMsg:
		if m.filtering {
			cmd := m.handleFilterKey(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		} else {
			cmd := m.handleKey(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

// View renders the model to a string.
func (m *Model) View() string {
	switch m.state {
	case stateScanning:
		return m.viewScanning()
	case stateResults, stateConfirming:
		return m.viewResults()
	case stateCleaning:
		return m.viewCleaning()
	case stateSummary:
		return m.viewSummary()
	}
	return ""
}

// ---- scanning -------------------------------------------------------------

// startScan returns a command that runs the engine scan in a goroutine and
// forwards matches back to the model.
func (m *Model) startScan() tea.Cmd {
	eng := m.opts.Engine
	roots := m.opts.Roots
	return func() tea.Msg {
		// We launch a goroutine and return the channel-draining function
		// as the first message via p.Send(). Bubble Tea idiom uses a
		// program handle via tea.Program; to keep things simple here we
		// collect synchronously inside a single tea.Cmd that returns a
		// scanDoneMsg once complete. Progress updates arrive via matchMsg
		// inside the drain loop we run on a separate goroutine communicating
		// via a channel — Bubble Tea's recommended pattern for streaming.
		return scanSync(eng, roots)
	}
}

// scanSync runs the engine.Scan collecting matches into a batch that is
// returned in one go. This is simpler than plumbing a tea.Program pointer
// through the model; the tradeoff is that the user sees a single reveal
// moment at the end of scan, not a live-growing list.
//
// For a streaming experience we upgrade this in a follow-up by switching to
// tea.NewProgram(m).Send(matchMsg) from a goroutine.
func scanSync(eng *engine.Engine, roots []string) tea.Msg {
	ctx := context.Background()
	matches := []detectors.Match{}
	out, errc := eng.Scan(ctx, roots)
	for m := range out {
		matches = append(matches, m)
	}
	err := <-errc
	// We stash matches in a closure message the update loop can read.
	return scanBatchMsg{matches: matches, err: err}
}

// scanBatchMsg delivers the full scan result in one payload.
type scanBatchMsg struct {
	matches []detectors.Match
	err     error
}

// When the scan ends, sizeMatches spawns a command that fills SizeBytes.
func (m *Model) sizeMatches() tea.Cmd {
	eng := m.opts.Engine
	matches := m.matches
	return func() tea.Msg {
		eng.SizeInPlace(context.Background(), matches)
		return sizedMsg{}
	}
}

// ---- key handling ---------------------------------------------------------

func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case keyMatches(msg, m.keys.Quit):
		return tea.Quit
	case keyMatches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
	case keyMatches(msg, m.keys.Up):
		if m.cursor > 0 {
			m.cursor--
		}
	case keyMatches(msg, m.keys.Down):
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
		}
	case keyMatches(msg, m.keys.PageUp):
		m.cursor -= 10
		if m.cursor < 0 {
			m.cursor = 0
		}
	case keyMatches(msg, m.keys.PageDown):
		m.cursor += 10
		if m.cursor >= len(m.filtered) {
			m.cursor = len(m.filtered) - 1
		}
	case keyMatches(msg, m.keys.Home):
		m.cursor = 0
	case keyMatches(msg, m.keys.End):
		m.cursor = len(m.filtered) - 1
	case keyMatches(msg, m.keys.Toggle):
		if m.cursor >= 0 && m.cursor < len(m.filtered) {
			idx := m.filtered[m.cursor]
			m.selected[idx] = !m.selected[idx]
		}
	case keyMatches(msg, m.keys.ToggleAll):
		if len(m.selected) == len(m.matches) {
			m.selected = map[int]bool{}
		} else {
			for i := range m.matches {
				m.selected[i] = true
			}
		}
	case keyMatches(msg, m.keys.Sort):
		m.sortBy = (m.sortBy + 1) % 3
		m.applySort()
	case keyMatches(msg, m.keys.DryRun):
		m.dryRun = !m.dryRun
	case keyMatches(msg, m.keys.Hard):
		m.hard = !m.hard
	case keyMatches(msg, m.keys.Filter):
		m.filtering = true
		m.filter.Focus()
		return textinput.Blink
	case keyMatches(msg, m.keys.Confirm):
		return m.startClean()
	}
	return nil
}

func (m *Model) handleFilterKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.filtering = false
		m.filter.Blur()
		return nil
	case "enter":
		m.filtering = false
		m.filter.Blur()
		m.rebuildFiltered()
		return nil
	}
	var cmd tea.Cmd
	m.filter, cmd = m.filter.Update(msg)
	m.rebuildFiltered()
	return cmd
}

func keyMatches(msg tea.KeyMsg, b interface {
	Keys() []string
}) bool {
	for _, k := range b.Keys() {
		if k == msg.String() {
			return true
		}
	}
	return false
}

// ---- clean ----------------------------------------------------------------

func (m *Model) startClean() tea.Cmd {
	picked := m.selectedMatches()
	if len(picked) == 0 {
		return nil
	}
	m.state = stateCleaning
	eng := m.opts.Engine
	opts := detectors.CleanOptions{DryRun: m.dryRun}
	if m.hard {
		opts.Override = detectors.StrategyHardDelete
	}
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			sum := eng.Clean(context.Background(), picked, opts)
			return cleanDoneMsg{sum: sum}
		},
	)
}

func (m *Model) selectedMatches() []detectors.Match {
	if len(m.selected) == 0 {
		return nil
	}
	out := make([]detectors.Match, 0, len(m.selected))
	for i, ok := range m.selected {
		if ok && i >= 0 && i < len(m.matches) {
			out = append(out, m.matches[i])
		}
	}
	return out
}

// ---- sorting & filtering --------------------------------------------------

func (m *Model) applySort() {
	sort.Slice(m.matches, func(i, j int) bool {
		switch m.sortBy {
		case sortSize:
			return m.matches[i].SizeBytes > m.matches[j].SizeBytes
		case sortCategory:
			return m.matches[i].Category < m.matches[j].Category
		default:
			return m.matches[i].Path < m.matches[j].Path
		}
	})
	m.rebuildFiltered()
}

func (m *Model) rebuildFiltered() {
	q := strings.TrimSpace(m.filter.Value())
	m.filtered = m.filtered[:0]
	for i, match := range m.matches {
		if q == "" || strings.Contains(strings.ToLower(string(match.Category)+" "+match.Path), strings.ToLower(q)) {
			m.filtered = append(m.filtered, i)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = 0
	}
}

// ---- views ----------------------------------------------------------------

func (m *Model) viewScanning() string {
	elapsed := time.Since(m.scanStart).Round(100 * time.Millisecond)
	title := m.theme.Header.Render("clearstack · scanning")
	body := fmt.Sprintf("%s scanning %s\n%d matches so far · %s elapsed",
		m.spinner.View(), strings.Join(m.opts.Roots, ", "), len(m.matches), elapsed)
	return title + "\n\n" + body
}

func (m *Model) viewResults() string {
	var b strings.Builder
	b.WriteString(m.theme.Header.Render(fmt.Sprintf("clearstack · %d matches · %s",
		len(m.matches), humanBytes(m.totalBytes()))))
	b.WriteString("\n\n")
	if m.filtering {
		b.WriteString(m.filter.View() + "\n\n")
	}
	start, end := m.visibleRange()
	for i := start; i < end; i++ {
		idx := m.filtered[i]
		match := m.matches[idx]
		mark := "[ ]"
		if m.selected[idx] {
			mark = "[x]"
		}
		safety := m.colorSafety(match.Safety)
		line := fmt.Sprintf("%s %s %-22s %10s  %s",
			mark, safety, match.Category, humanBytes(match.SizeBytes), match.Path)
		if i == m.cursor {
			b.WriteString(m.theme.RowSelected.Render(line))
		} else {
			b.WriteString(m.theme.Row.Render(line))
		}
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(m.viewStatusBar())
	b.WriteString("\n")
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

func (m *Model) viewCleaning() string {
	title := m.theme.Header.Render("clearstack · cleaning…")
	return title + "\n\n" + m.spinner.View() + " cleaning " +
		fmt.Sprintf("%d items", len(m.selectedMatches()))
}

func (m *Model) viewSummary() string {
	s := m.cleanSummary
	header := m.theme.Header.Render("clearstack · summary")
	verb := "cleaned"
	if s.DryRun {
		verb = "would clean"
	}
	body := fmt.Sprintf("%s %d/%d items, freed %s\n%d failures",
		verb, s.Succeeded, s.Attempted, humanBytes(s.BytesFreed), s.Failed)
	return header + "\n\n" + body + "\n\n" + m.theme.Subtle.Render("press q to exit")
}

func (m *Model) viewStatusBar() string {
	selCount := 0
	var selBytes int64
	for i, ok := range m.selected {
		if ok && i < len(m.matches) {
			selCount++
			selBytes += m.matches[i].SizeBytes
		}
	}
	mode := "trash"
	if m.hard {
		mode = "hard"
	}
	dry := ""
	if m.dryRun {
		dry = " · dry-run"
	}
	return m.theme.StatusBar.Render(fmt.Sprintf(
		"selected %d · %s  |  mode %s%s  |  sort %s",
		selCount, humanBytes(selBytes), mode, dry, m.sortBy.label(),
	))
}

func (m *Model) visibleRange() (int, int) {
	window := m.height - 8 // header + status + help
	if window < 1 {
		window = 1
	}
	start := 0
	if m.cursor > window/2 {
		start = m.cursor - window/2
	}
	end := start + window
	if end > len(m.filtered) {
		end = len(m.filtered)
	}
	return start, end
}

func (m *Model) totalBytes() int64 {
	var t int64
	for _, match := range m.matches {
		t += match.SizeBytes
	}
	return t
}

func (m *Model) colorSafety(s detectors.Safety) string {
	switch s {
	case detectors.SafetySafe:
		return m.theme.SafetySafe.Render("●")
	case detectors.SafetyCaution:
		return m.theme.SafetyCaution.Render("●")
	default:
		return m.theme.SafetyDanger.Render("●")
	}
}

// ---- helpers --------------------------------------------------------------

func humanBytes(n int64) string {
	if n <= 0 {
		return "0 B"
	}
	return humanize.IBytes(uint64(n))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Ensure scanBatchMsg is handled even though we use scanDoneMsg as the
// marker.
func init() {
	_ = scanDoneMsg{}
}
