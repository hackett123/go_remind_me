package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
)

// keyMap defines all key bindings
type keyMap struct {
	Up            key.Binding
	Down          key.Binding
	Acknowledge   key.Binding
	Unacknowledge key.Binding
	Delete        key.Binding
	Snooze5m      key.Binding
	Snooze1h      key.Binding
	Snooze1d      key.Binding
	Filter        key.Binding
	Add           key.Binding
	Edit          key.Binding
	Detail        key.Binding
	Theme         key.Binding
	Layout        key.Binding
	Sort          key.Binding
	Help          key.Binding
	Quit          key.Binding
}

// ShortHelp returns key bindings for the short help view
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Acknowledge, k.Filter, k.Add, k.Help, k.Quit}
}

// FullHelp returns key bindings for the full help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Acknowledge, k.Unacknowledge},
		{k.Snooze5m, k.Snooze1h, k.Snooze1d, k.Delete},
		{k.Filter, k.Add, k.Edit, k.Detail, k.Theme, k.Layout, k.Sort, k.Help, k.Quit},
	}
}

var _ help.KeyMap = keyMap{}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Acknowledge: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter", "done"),
	),
	Unacknowledge: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "unack"),
	),
	Delete: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("dd", "delete"),
	),
	Snooze5m: key.NewBinding(
		key.WithKeys("1"),
		key.WithHelp("1", "snooze 5m"),
	),
	Snooze1h: key.NewBinding(
		key.WithKeys("2"),
		key.WithHelp("2", "snooze 1h"),
	),
	Snooze1d: key.NewBinding(
		key.WithKeys("3"),
		key.WithHelp("3", "snooze 1d"),
	),
	Filter: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "filter"),
	),
	Add: key.NewBinding(
		key.WithKeys("n"),
		key.WithHelp("n", "new"),
	),
	Edit: key.NewBinding(
		key.WithKeys("e"),
		key.WithHelp("e", "edit"),
	),
	Detail: key.NewBinding(
		key.WithKeys("K"),
		key.WithHelp("K", "detail"),
	),
	Theme: key.NewBinding(
		key.WithKeys("t"),
		key.WithHelp("t", "theme"),
	),
	Layout: key.NewBinding(
		key.WithKeys("v"),
		key.WithHelp("v", "view"),
	),
	Sort: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "sort"),
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
