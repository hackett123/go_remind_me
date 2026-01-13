package tui

import "github.com/charmbracelet/lipgloss"

type Theme struct {
	Name        string
	Title       lipgloss.Color
	Normal      lipgloss.Color
	Triggered   lipgloss.Color
	Acknowledged lipgloss.Color
	Source      lipgloss.Color
	Selected    lipgloss.Color
	Accent      lipgloss.Color
	Muted       lipgloss.Color
}

var themes = []Theme{
	{
		Name:        "Everforest",
		Title:       lipgloss.Color("#A7C080"),
		Normal:      lipgloss.Color("#D3C6AA"),
		Triggered:   lipgloss.Color("#E67E80"),
		Acknowledged: lipgloss.Color("#859289"),
		Source:      lipgloss.Color("#859289"),
		Selected:    lipgloss.Color("#83C092"),
		Accent:      lipgloss.Color("#7FBBB3"),
		Muted:       lipgloss.Color("#7A8478"),
	},
	{
		Name:        "Kiro Purple",
		Title:       lipgloss.Color("205"),
		Normal:      lipgloss.Color("252"),
		Triggered:   lipgloss.Color("196"),
		Acknowledged: lipgloss.Color("241"),
		Source:      lipgloss.Color("241"),
		Selected:    lipgloss.Color("170"),
		Accent:      lipgloss.Color("205"),
		Muted:       lipgloss.Color("241"),
	},
	{
		Name:        "Dracula",
		Title:       lipgloss.Color("#bd93f9"),
		Normal:      lipgloss.Color("#f8f8f2"),
		Triggered:   lipgloss.Color("#ff5555"),
		Acknowledged: lipgloss.Color("#6272a4"),
		Source:      lipgloss.Color("#6272a4"),
		Selected:    lipgloss.Color("#50fa7b"),
		Accent:      lipgloss.Color("#ff79c6"),
		Muted:       lipgloss.Color("#6272a4"),
	},
	{
		Name:        "Nord",
		Title:       lipgloss.Color("#88c0d0"),
		Normal:      lipgloss.Color("#eceff4"),
		Triggered:   lipgloss.Color("#bf616a"),
		Acknowledged: lipgloss.Color("#4c566a"),
		Source:      lipgloss.Color("#4c566a"),
		Selected:    lipgloss.Color("#a3be8c"),
		Accent:      lipgloss.Color("#81a1c1"),
		Muted:       lipgloss.Color("#4c566a"),
	},
	{
		Name:        "Solarized",
		Title:       lipgloss.Color("#268bd2"),
		Normal:      lipgloss.Color("#839496"),
		Triggered:   lipgloss.Color("#dc322f"),
		Acknowledged: lipgloss.Color("#586e75"),
		Source:      lipgloss.Color("#586e75"),
		Selected:    lipgloss.Color("#859900"),
		Accent:      lipgloss.Color("#2aa198"),
		Muted:       lipgloss.Color("#586e75"),
	},
	{
		Name:        "Monokai",
		Title:       lipgloss.Color("#f92672"),
		Normal:      lipgloss.Color("#f8f8f2"),
		Triggered:   lipgloss.Color("#f92672"),
		Acknowledged: lipgloss.Color("#75715e"),
		Source:      lipgloss.Color("#75715e"),
		Selected:    lipgloss.Color("#a6e22e"),
		Accent:      lipgloss.Color("#66d9ef"),
		Muted:       lipgloss.Color("#75715e"),
	},
}

func (t Theme) applyStyles() {
	titleStyle = lipgloss.NewStyle().
		Foreground(t.Title).
		Bold(true).
		MarginLeft(2)

	normalStyle = lipgloss.NewStyle().
		Foreground(t.Normal)

	triggeredStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(t.Triggered)

	acknowledgedStyle = lipgloss.NewStyle().
		Foreground(t.Acknowledged).
		Strikethrough(true)

	sourceStyle = lipgloss.NewStyle().
		Foreground(t.Source)

	selectedItemStyle = lipgloss.NewStyle().
		Foreground(t.Selected).
		Bold(true)

	inputBoxStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(t.Accent).
		Padding(0, 1).
		MarginTop(1).
		MarginBottom(1)

	inputLabelStyle = lipgloss.NewStyle().
		Foreground(t.Accent).
		Bold(true)

	inputHintStyle = lipgloss.NewStyle().
		Foreground(t.Muted).
		Italic(true)

	welcomeTitleStyle = lipgloss.NewStyle().
		Foreground(t.Title).
		Bold(true).
		MarginBottom(1)

	welcomeTextStyle = lipgloss.NewStyle().
		Foreground(t.Normal)

	welcomeHighlightStyle = lipgloss.NewStyle().
		Foreground(t.Selected).
		Bold(true)
}
