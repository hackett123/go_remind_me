package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"go_remind/reminder"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("241"))

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Background(lipgloss.Color("237"))

	triggeredStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196")) // Red

	triggeredSelectedStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("196")).
				Background(lipgloss.Color("237"))

	acknowledgedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Strikethrough(true)

	acknowledgedSelectedStyle = lipgloss.NewStyle().
					Foreground(lipgloss.Color("241")).
					Strikethrough(true).
					Background(lipgloss.Color("237"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)
)

// TickMsg is sent every second to check for triggered reminders
type TickMsg time.Time

// FileUpdateMsg is sent when a watched file is updated
type FileUpdateMsg struct {
	FilePath  string
	Reminders []*reminder.Reminder
}

// Model is the Bubble Tea model for the reminder TUI
type Model struct {
	reminders     []*reminder.Reminder
	cursor        int
	width         int
	height        int
	watcherEvents <-chan FileUpdateMsg // Channel for file update events
}

// New creates a new TUI model with the given reminders
func New(reminders []*reminder.Reminder, watcherEvents <-chan FileUpdateMsg) Model {
	return Model{
		reminders:     reminders,
		cursor:        0,
		watcherEvents: watcherEvents,
	}
}

// Init initializes the model and starts the tick timer
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tickCmd(),
		tea.EnterAltScreen,
	}
	if m.watcherEvents != nil {
		cmds = append(cmds, m.waitForFileUpdate())
	}
	return tea.Batch(cmds...)
}

// tickCmd returns a command that sends a TickMsg after 1 second
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// waitForFileUpdate waits for a file update event from the watcher
func (m Model) waitForFileUpdate() tea.Cmd {
	return func() tea.Msg {
		if m.watcherEvents == nil {
			return nil
		}
		event, ok := <-m.watcherEvents
		if !ok {
			return nil
		}
		return event
	}
}

// snooze postpones the currently selected triggered reminder by the given duration
func (m *Model) snooze(duration time.Duration) {
	if len(m.reminders) == 0 {
		return
	}
	r := m.reminders[m.cursor]
	if r.Status != reminder.Triggered {
		return
	}
	r.DateTime = time.Now().Add(duration)
	r.Status = reminder.Pending
	reminder.SortByDateTime(m.reminders)
	// Adjust cursor to follow the snoozed reminder or stay in bounds
	if m.cursor >= len(m.reminders) {
		m.cursor = len(m.reminders) - 1
	}
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Up):
			if m.cursor > 0 {
				m.cursor--
			}

		case key.Matches(msg, keys.Down):
			if m.cursor < len(m.reminders)-1 {
				m.cursor++
			}

		case key.Matches(msg, keys.Acknowledge):
			if len(m.reminders) > 0 {
				r := m.reminders[m.cursor]
				if r.Status == reminder.Triggered {
					r.Status = reminder.Acknowledged
				}
			}

		case key.Matches(msg, keys.Snooze5m):
			m.snooze(5 * time.Minute)

		case key.Matches(msg, keys.Snooze1h):
			m.snooze(1 * time.Hour)

		case key.Matches(msg, keys.Snooze1d):
			m.snooze(24 * time.Hour)
		}

	case TickMsg:
		// Check for newly triggered reminders
		now := time.Now()
		for _, r := range m.reminders {
			if r.Status == reminder.Pending && now.After(r.DateTime) {
				r.Status = reminder.Triggered
			}
		}
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case FileUpdateMsg:
		// Merge new reminders with existing ones (deduplication)
		m.reminders = reminder.MergeFromFile(m.reminders, msg.FilePath, msg.Reminders)
		reminder.SortByDateTime(m.reminders)
		// Keep cursor in bounds
		if m.cursor >= len(m.reminders) && len(m.reminders) > 0 {
			m.cursor = len(m.reminders) - 1
		}
		// Continue listening for more updates
		return m, m.waitForFileUpdate()
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Title
	b.WriteString(titleStyle.Render("Go Remind"))
	b.WriteString("\n\n")

	// Header
	header := fmt.Sprintf("%-20s %-12s %s", "TIME", "STATUS", "DESCRIPTION")
	b.WriteString(headerStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(headerStyle.Render(strings.Repeat("─", 60)))
	b.WriteString("\n")

	// Reminders
	if len(m.reminders) == 0 {
		b.WriteString(normalStyle.Render("No reminders found."))
		b.WriteString("\n")
	} else {
		for i, r := range m.reminders {
			line := formatReminder(r)
			style := getStyle(r, i == m.cursor)
			b.WriteString(style.Render(line))
			b.WriteString("\n")
		}
	}

	// Help
	b.WriteString(helpStyle.Render("↑/↓: navigate  enter: acknowledge  1/2/3: snooze 5m/1h/1d  q: quit"))

	return b.String()
}

// formatReminder formats a single reminder as a table row
func formatReminder(r *reminder.Reminder) string {
	timeStr := r.DateTime.Format("Jan 2 3:04pm")
	return fmt.Sprintf("%-20s %-12s %s", timeStr, r.Status.String(), r.Description)
}

// getStyle returns the appropriate style for a reminder
func getStyle(r *reminder.Reminder, selected bool) lipgloss.Style {
	switch r.Status {
	case reminder.Triggered:
		if selected {
			return triggeredSelectedStyle
		}
		return triggeredStyle
	case reminder.Acknowledged:
		if selected {
			return acknowledgedSelectedStyle
		}
		return acknowledgedStyle
	default:
		if selected {
			return selectedStyle
		}
		return normalStyle
	}
}

// Key bindings
type keyMap struct {
	Up          key.Binding
	Down        key.Binding
	Acknowledge key.Binding
	Snooze5m    key.Binding
	Snooze1h    key.Binding
	Snooze1d    key.Binding
	Quit        key.Binding
}

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
		key.WithHelp("enter/space", "acknowledge"),
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
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
