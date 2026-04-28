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
	// stateCategories is the first screen the user sees: a checkbox list of
	// every detector registered for the current platform. The user picks
	// which categories to include before any disk scan happens. Default
	// selection is "all SafetySafe categories" — caution and danger
	// detectors require an explicit opt-in, mirroring the CLI semantics.
	stateCategories state = iota
	stateScanning
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

	// Category picker state (stateCategories)
	categories     []detectors.Detector
	categoryPicked map[detectors.Category]bool
	categoryCursor int

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

	// dangerAcknowledged is the explicit opt-in required on stateConfirming
	// when the current selection includes any [danger] category. It resets
	// every time the user enters stateConfirming so a typo on a previous
	// confirmation cannot leak into the next one.
	dangerAcknowledged bool
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

	cats := opts.Engine.EnabledCategories()
	picked := make(map[detectors.Category]bool, len(cats))
	for _, d := range cats {
		// Default: every Safe category is on; Caution/Danger require opt-in.
		picked[d.ID()] = d.Safety() == detectors.SafetySafe
	}

	return &Model{
		opts:           opts,
		state:          stateCategories,
		theme:          DefaultTheme(),
		keys:           DefaultKeys(),
		help:           help.New(),
		spinner:        sp,
		progress:       pg,
		filter:         fi,
		selected:       make(map[int]bool),
		sortBy:         sortSize,
		categories:     cats,
		categoryPicked: picked,
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

// Init starts the spinner. The first scan does NOT begin automatically —
// the user picks categories on stateCategories, presses enter to confirm,
// and the model transitions into stateScanning at that point.
func (m *Model) Init() tea.Cmd {
	return m.spinner.Tick
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

	case scanBatchMsg:
		m.matches = append(m.matches, msg.matches...)
		m.scanDone = true
		m.scanErr = msg.err
		m.rebuildFiltered()
		cmds = append(cmds, m.sizeMatches())

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
		switch {
		case m.state == stateCategories:
			if cmd := m.handleCategoryKey(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case m.state == stateConfirming:
			if cmd := m.handleConfirmKey(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		case m.filtering:
			if cmd := m.handleFilterKey(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		default:
			if cmd := m.handleKey(msg); cmd != nil {
				cmds = append(cmds, cmd)
			}
		}
	}
	return m, tea.Batch(cmds...)
}

// View renders the model to a string.
func (m *Model) View() string {
	switch m.state {
	case stateCategories:
		return m.viewCategories()
	case stateScanning:
		return m.viewScanning()
	case stateResults:
		return m.viewResults()
	case stateConfirming:
		return m.viewConfirming()
	case stateCleaning:
		return m.viewCleaning()
	case stateSummary:
		return m.viewSummary()
	}
	return ""
}

// ---- scanning -------------------------------------------------------------

// startScan returns a command that runs the engine scan in a goroutine and
// forwards matches back to the model. picked is the explicit category list
// the user confirmed on the picker — it is also used to synthesize matches
// for scan-inert detectors (Docker prune targets, etc).
func (m *Model) startScan(picked []detectors.Category) tea.Cmd {
	eng := m.opts.Engine
	roots := m.opts.Roots
	return func() tea.Msg {
		return scanSync(eng, roots, picked)
	}
}

// scanSync runs engine.Scan, then appends synthetic matches for any
// scan-inert detectors the user picked (e.g., docker_build_cache). The full
// batch is returned in one tea message; the user sees a single reveal at
// the end of the scan rather than a live-growing list.
func scanSync(eng *engine.Engine, roots []string, picked []detectors.Category) tea.Msg {
	ctx := context.Background()
	matches := []detectors.Match{}
	out, errc := eng.Scan(ctx, roots)
	for m := range out {
		matches = append(matches, m)
	}
	err := <-errc

	// Synthesize matches for scan-inert categories (docker_*) the user
	// picked. Without this, picking docker_build_cache in the TUI would
	// silently behave the same as not picking it.
	seen := make(map[detectors.Category]struct{}, len(matches))
	for _, mm := range matches {
		seen[mm.Category] = struct{}{}
	}
	for _, id := range picked {
		if _, dup := seen[id]; dup {
			continue
		}
		d := detectors.Default.Get(id)
		if d == nil || !d.PlatformSupported() {
			continue
		}
		syn, ok := d.(detectors.Synthesizer)
		if !ok {
			continue
		}
		if mm := syn.Synthesize(); mm != nil {
			matches = append(matches, *mm)
		}
	}

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
	case keyMatches(msg, m.keys.BackToPicker):
		// Discard scan/selection state and re-render the category picker.
		m.matches = nil
		m.filtered = nil
		m.selected = make(map[int]bool)
		m.cursor = 0
		m.scanDone = false
		m.scanErr = nil
		m.state = stateCategories
	case keyMatches(msg, m.keys.Confirm):
		// Enter on stateResults moves to the confirmation screen (where
		// the user reviews count/size/mode and any [danger] selections).
		// Actually starting the cleanup is gated by Y on stateConfirming.
		if len(m.selectedMatches()) == 0 {
			return nil
		}
		m.dangerAcknowledged = false
		m.state = stateConfirming
	}
	return nil
}

// handleConfirmKey drives the stateConfirming screen — the explicit Y/N
// gate between picking matches and actually running cleanup. The user can:
//
//   - press Y to commit (blocked by the danger guard if any [danger]
//     category is in the selection and `D` has not been pressed first),
//   - press D to acknowledge danger items (toggles m.dangerAcknowledged),
//   - press N or esc to return to stateResults and adjust the selection.
func (m *Model) handleConfirmKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "y", "Y":
		if m.hasDangerSelection() && !m.dangerAcknowledged {
			// Block until user acknowledges danger items explicitly.
			return nil
		}
		return m.startClean()
	case "d", "D":
		m.dangerAcknowledged = !m.dangerAcknowledged
	case "n", "N", "esc":
		m.state = stateResults
	case "q", "ctrl+c":
		return tea.Quit
	}
	return nil
}

// handleCategoryKey drives the stateCategories screen. The user can move
// the cursor, toggle individual entries with space, flip every entry with
// `a`, snap to "safe-only" with `S`, and press enter to lock in the
// selection and start the scan.
func (m *Model) handleCategoryKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case keyMatches(msg, m.keys.Quit):
		return tea.Quit
	case keyMatches(msg, m.keys.Help):
		m.help.ShowAll = !m.help.ShowAll
	case keyMatches(msg, m.keys.Up):
		if m.categoryCursor > 0 {
			m.categoryCursor--
		}
	case keyMatches(msg, m.keys.Down):
		if m.categoryCursor < len(m.categories)-1 {
			m.categoryCursor++
		}
	case keyMatches(msg, m.keys.PageUp):
		m.categoryCursor -= 10
		if m.categoryCursor < 0 {
			m.categoryCursor = 0
		}
	case keyMatches(msg, m.keys.PageDown):
		m.categoryCursor += 10
		if m.categoryCursor >= len(m.categories) {
			m.categoryCursor = len(m.categories) - 1
		}
	case keyMatches(msg, m.keys.Home):
		m.categoryCursor = 0
	case keyMatches(msg, m.keys.End):
		m.categoryCursor = len(m.categories) - 1
	case keyMatches(msg, m.keys.Toggle):
		if m.categoryCursor >= 0 && m.categoryCursor < len(m.categories) {
			id := m.categories[m.categoryCursor].ID()
			m.categoryPicked[id] = !m.categoryPicked[id]
		}
	case keyMatches(msg, m.keys.ToggleAll):
		// If everything is on, turn everything off; otherwise turn
		// everything on. This matches the results screen ToggleAll.
		allOn := true
		for _, d := range m.categories {
			if !m.categoryPicked[d.ID()] {
				allOn = false
				break
			}
		}
		for _, d := range m.categories {
			m.categoryPicked[d.ID()] = !allOn
		}
	case keyMatches(msg, m.keys.OnlySafe):
		for _, d := range m.categories {
			m.categoryPicked[d.ID()] = d.Safety() == detectors.SafetySafe
		}
	case keyMatches(msg, m.keys.Confirm):
		return m.commitCategoriesAndScan()
	}
	return nil
}

// commitCategoriesAndScan applies the user's category picks to the engine
// and transitions to stateScanning. It rejects an empty selection by
// staying on the picker — scanning with zero detectors finds nothing.
func (m *Model) commitCategoriesAndScan() tea.Cmd {
	picked := make([]detectors.Category, 0, len(m.categoryPicked))
	for id, on := range m.categoryPicked {
		if on {
			picked = append(picked, id)
		}
	}
	if len(picked) == 0 {
		return nil
	}
	m.opts.Engine.SetCategories(picked)
	m.state = stateScanning
	m.scanStart = time.Now()
	return tea.Batch(m.spinner.Tick, m.startScan(picked))
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

// hasDangerSelection reports whether any currently-selected match belongs
// to a [danger] category. The confirmation screen consults this to decide
// whether to require an explicit `D` acknowledgement before unlocking Y.
func (m *Model) hasDangerSelection() bool {
	for _, mm := range m.selectedMatches() {
		if mm.Safety == detectors.SafetyDanger {
			return true
		}
	}
	return false
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

// viewCategories renders the initial category-picker screen. Categories
// are listed alphabetically (the engine returns them sorted), with the
// safety dot in front of each id. The status bar reports how many
// categories are selected and the help line lists the relevant keys.
func (m *Model) viewCategories() string {
	var b strings.Builder
	pickedCount := 0
	for _, on := range m.categoryPicked {
		if on {
			pickedCount++
		}
	}
	b.WriteString(m.theme.Header.Render(fmt.Sprintf(
		"clearstack · pick categories  (%d/%d selected)",
		pickedCount, len(m.categories),
	)))
	b.WriteString("\n\n")

	start, end := m.categoryVisibleRange()
	for i := start; i < end; i++ {
		d := m.categories[i]
		mark := "[ ]"
		if m.categoryPicked[d.ID()] {
			mark = "[x]"
		}
		safety := m.colorSafety(d.Safety())
		line := fmt.Sprintf("%s %s %-22s  %s", mark, safety, d.ID(), d.Description())
		if i == m.categoryCursor {
			b.WriteString(m.theme.RowSelected.Render(line))
		} else {
			b.WriteString(m.theme.Row.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(m.theme.StatusBar.Render(fmt.Sprintf(
		"selected %d/%d  ·  roots: %s  ·  enter to scan",
		pickedCount, len(m.categories), strings.Join(m.opts.Roots, ", "),
	)))
	b.WriteString("\n")
	b.WriteString(m.help.View(m.keys))
	return b.String()
}

func (m *Model) categoryVisibleRange() (int, int) {
	window := m.height - 6
	if window < 1 {
		window = 1
	}
	start := 0
	if m.categoryCursor > window/2 {
		start = m.categoryCursor - window/2
	}
	end := start + window
	if end > len(m.categories) {
		end = len(m.categories)
	}
	return start, end
}

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

// viewConfirming renders the explicit Y/N gate the user must clear before
// any cleanup runs. It reports the count, total size, mode (trash/hard,
// dry-run flag) and — when applicable — calls out [danger] selections in
// red and demands an explicit `D` acknowledgement before unlocking Y.
//
// This is the last opportunity to bail out (N or esc returns to results).
func (m *Model) viewConfirming() string {
	picked := m.selectedMatches()

	var totalBytes int64
	dangerSelected := make([]detectors.Match, 0)
	for _, p := range picked {
		totalBytes += p.SizeBytes
		if p.Safety == detectors.SafetyDanger {
			dangerSelected = append(dangerSelected, p)
		}
	}

	var b strings.Builder
	b.WriteString(m.theme.Header.Render("clearstack · confirm cleanup"))
	b.WriteString("\n\n")

	mode := "trash (recoverable via clearstack undo)"
	if m.hard {
		mode = "HARD DELETE (irreversible — bypasses Lixeira)"
	}
	dryRun := ""
	if m.dryRun {
		dryRun = "  [dry-run: no changes will be made]"
	}

	b.WriteString(fmt.Sprintf("about to clean %d items totalling %s\n",
		len(picked), humanBytes(totalBytes)))
	b.WriteString(fmt.Sprintf("mode: %s%s\n\n", mode, dryRun))

	if len(dangerSelected) > 0 {
		b.WriteString(m.theme.SafetyDanger.Render(
			fmt.Sprintf("⚠  %d DANGER item(s) selected — they may DESTROY DATA:",
				len(dangerSelected))))
		b.WriteString("\n")
		for _, d := range dangerSelected {
			b.WriteString(m.theme.SafetyDanger.Render("  • "+string(d.Category)+" → "+d.Path) + "\n")
		}
		b.WriteString("\n")
		if m.dangerAcknowledged {
			b.WriteString(m.theme.SafetyCaution.Render("[D] danger acknowledged — Y is unlocked"))
		} else {
			b.WriteString(m.theme.SafetyDanger.Render(
				"press D to acknowledge (required before Y will run cleanup)"))
		}
		b.WriteString("\n\n")
	}

	prompt := "[Y] yes, clean   [N] no, go back   [esc] back to results   [q] quit"
	b.WriteString(m.theme.StatusBar.Render(prompt))
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
