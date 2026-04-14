package tui

import "github.com/charmbracelet/lipgloss"

// Theme groups every style the TUI draws.
type Theme struct {
	Base          lipgloss.Style
	Header        lipgloss.Style
	Subtle        lipgloss.Style
	Accent        lipgloss.Style
	Danger        lipgloss.Style
	Warning       lipgloss.Style
	SafetySafe    lipgloss.Style
	SafetyCaution lipgloss.Style
	SafetyDanger  lipgloss.Style
	SelectedRow   lipgloss.Style
	Row           lipgloss.Style
	RowSelected   lipgloss.Style
	StatusBar     lipgloss.Style
	StatusBarDim  lipgloss.Style
	SpinnerDot    lipgloss.Style
}

// Catppuccin Mocha palette.
var (
	colRosewater = lipgloss.Color("#f5e0dc")
	colMauve     = lipgloss.Color("#cba6f7")
	colRed       = lipgloss.Color("#f38ba8")
	colPeach     = lipgloss.Color("#fab387")
	colYellow    = lipgloss.Color("#f9e2af")
	colGreen     = lipgloss.Color("#a6e3a1")
	colTeal      = lipgloss.Color("#94e2d5")
	colBlue      = lipgloss.Color("#89b4fa")
	colOverlay0  = lipgloss.Color("#6c7086")
	colSurface0  = lipgloss.Color("#313244")
	colBase      = lipgloss.Color("#1e1e2e")
	colText      = lipgloss.Color("#cdd6f4")
)

// DefaultTheme returns the Catppuccin Mocha-based dark theme.
func DefaultTheme() Theme {
	return Theme{
		Base:          lipgloss.NewStyle().Foreground(colText),
		Header:        lipgloss.NewStyle().Bold(true).Foreground(colMauve).Padding(0, 1),
		Subtle:        lipgloss.NewStyle().Foreground(colOverlay0),
		Accent:        lipgloss.NewStyle().Foreground(colBlue),
		Danger:        lipgloss.NewStyle().Foreground(colRed).Bold(true),
		Warning:       lipgloss.NewStyle().Foreground(colPeach),
		SafetySafe:    lipgloss.NewStyle().Foreground(colGreen),
		SafetyCaution: lipgloss.NewStyle().Foreground(colYellow),
		SafetyDanger:  lipgloss.NewStyle().Foreground(colRed),
		Row:           lipgloss.NewStyle().Foreground(colText).Padding(0, 1),
		RowSelected:   lipgloss.NewStyle().Foreground(colBase).Background(colMauve).Bold(true).Padding(0, 1),
		SelectedRow:   lipgloss.NewStyle().Foreground(colRosewater).Padding(0, 1),
		StatusBar:     lipgloss.NewStyle().Foreground(colText).Background(colSurface0).Padding(0, 1),
		StatusBarDim:  lipgloss.NewStyle().Foreground(colOverlay0).Background(colSurface0).Padding(0, 1),
		SpinnerDot:    lipgloss.NewStyle().Foreground(colTeal),
	}
}
