package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap binds every interactive action in the TUI. Exported so the help
// component can render a consistent overlay.
type KeyMap struct {
	Up           key.Binding
	Down         key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	Home         key.Binding
	End          key.Binding
	Toggle       key.Binding
	ToggleAll    key.Binding
	OnlySafe     key.Binding
	Filter       key.Binding
	Sort         key.Binding
	DryRun       key.Binding
	Hard         key.Binding
	Confirm      key.Binding
	Cancel       key.Binding
	BackToPicker key.Binding
	Help         key.Binding
	Quit         key.Binding
}

// DefaultKeys returns the default clearstack keybindings — a blend of
// npkill and vim flavors.
func DefaultKeys() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+u"),
			key.WithHelp("pgup", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+d"),
			key.WithHelp("pgdn", "page down"),
		),
		Home: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("g", "top"),
		),
		End: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("G", "bottom"),
		),
		Toggle: key.NewBinding(
			key.WithKeys(" ", "x"),
			key.WithHelp("space", "toggle select"),
		),
		ToggleAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "toggle all"),
		),
		OnlySafe: key.NewBinding(
			key.WithKeys("S"),
			key.WithHelp("S", "select safe only"),
		),
		BackToPicker: key.NewBinding(
			key.WithKeys("backspace", "esc"),
			key.WithHelp("←/esc", "back to category picker"),
		),
		Filter: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "filter"),
		),
		Sort: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "sort"),
		),
		DryRun: key.NewBinding(
			key.WithKeys("D"),
			key.WithHelp("D", "dry-run"),
		),
		Hard: key.NewBinding(
			key.WithKeys("H"),
			key.WithHelp("H", "hard delete"),
		),
		Confirm: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "clean selected"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns the compact bottom help line.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Toggle, k.ToggleAll, k.Filter, k.Sort, k.Confirm, k.Help, k.Quit}
}

// FullHelp returns grouped help for the ? overlay.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.PageUp, k.PageDown, k.Home, k.End},
		{k.Toggle, k.ToggleAll, k.OnlySafe, k.Filter, k.Sort},
		{k.DryRun, k.Hard, k.Confirm, k.Cancel, k.BackToPicker},
		{k.Help, k.Quit},
	}
}
