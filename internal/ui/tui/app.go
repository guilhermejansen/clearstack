package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Run starts a Bubble Tea program with a fresh Model and blocks until the
// user exits. It is the single entry point the cmd/clearstack package uses.
func Run(opts Options) error {
	m := New(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
