package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"go_remind/reminder"
)

// Styles
var (
	appStyle = lipgloss.NewStyle().Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginLeft(2)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	triggeredStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("196"))

	acknowledgedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241")).
				Strikethrough(true)

	sourceStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("170")).
				Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1).
			MarginLeft(2)
)

// TickMsg is sent every second to check for triggered reminders
type TickMsg time.Time

// FileUpdateMsg is sent when a watched file is updated
type FileUpdateMsg struct {
	FilePath  string
	Reminders []*reminder.Reminder
}

// reminderItem wraps a Reminder to implement list.Item
type reminderItem struct {
	reminder *reminder.Reminder
}

func (i reminderItem) Title() string {
	return i.reminder.Description
}

func (i reminderItem) Description() string {
	return i.reminder.DateTime.Format("Jan 2 3:04pm")
}

func (i reminderItem) FilterValue() string {
	return i.reminder.Description
}

// itemDelegate handles rendering of list items
type itemDelegate struct{}

func (d itemDelegate) Height() int                             { return 1 }
func (d itemDelegate) Spacing() int                            { return 0 }
func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(reminderItem)
	if !ok {
		return
	}

	r := i.reminder
	timeStr := r.DateTime.Format("Jan 2 3:04pm")
	source := filepath.Base(r.SourceFile)

	// Build the line content
	var statusIcon string
	var style lipgloss.Style

	switch r.Status {
	case reminder.Triggered:
		statusIcon = "🔔"
		style = triggeredStyle
	case reminder.Acknowledged:
		statusIcon = "✓"
		style = acknowledgedStyle
	default:
		statusIcon = "○"
		style = normalStyle
	}

	isSelected := index == m.Index()
	if isSelected {
		statusIcon = "▸"
		if r.Status != reminder.Triggered && r.Status != reminder.Acknowledged {
			style = selectedItemStyle
		}
	}

	// Truncate description if needed
	desc := r.Description
	if len(desc) > 35 {
		desc = desc[:32] + "..."
	}

	line := fmt.Sprintf("%s %-18s %-12s %-36s", statusIcon, timeStr, r.Status.String(), desc)
	styledLine := style.Render(line)
	sourcePart := sourceStyle.Render(source)

	fmt.Fprintf(w, "%s  %s", styledLine, sourcePart)
}

// Model is the Bubble Tea model for the reminder TUI
type Model struct {
	list          list.Model
	reminders     []*reminder.Reminder
	watcherEvents <-chan FileUpdateMsg
	pendingDelete bool
	width         int
	height        int
}

// New creates a new TUI model with the given reminders
func New(reminders []*reminder.Reminder, watcherEvents <-chan FileUpdateMsg) Model {
	items := remindersToItems(reminders)

	l := list.New(items, itemDelegate{}, 80, 20)
	l.Title = "Go Remind"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false) // We'll show our own help

	return Model{
		list:          l,
		reminders:     reminders,
		watcherEvents: watcherEvents,
	}
}

func remindersToItems(reminders []*reminder.Reminder) []list.Item {
	items := make([]list.Item, len(reminders))
	for i, r := range reminders {
		items[i] = reminderItem{reminder: r}
	}
	return items
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

// refreshList updates the list items from the current reminders
func (m *Model) refreshList() {
	items := remindersToItems(m.reminders)
	m.list.SetItems(items)
}

// selectedReminder returns the currently selected reminder, or nil if none
func (m *Model) selectedReminder() *reminder.Reminder {
	if len(m.reminders) == 0 {
		return nil
	}
	idx := m.list.Index()
	if idx < 0 || idx >= len(m.reminders) {
		return nil
	}
	return m.reminders[idx]
}

// snooze postpones the currently selected triggered reminder by the given duration
func (m *Model) snooze(duration time.Duration) {
	r := m.selectedReminder()
	if r == nil || r.Status != reminder.Triggered {
		return
	}
	r.DateTime = time.Now().Add(duration)
	r.Status = reminder.Pending
	reminder.SortByDateTime(m.reminders)
	m.refreshList()
}

// deleteCurrentReminder removes the currently selected reminder from tracking
func (m *Model) deleteCurrentReminder() {
	if len(m.reminders) == 0 {
		return
	}
	idx := m.list.Index()
	if idx < 0 || idx >= len(m.reminders) {
		return
	}
	m.reminders = append(m.reminders[:idx], m.reminders[idx+1:]...)
	m.refreshList()
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't handle keys if filtering
		if m.list.FilterState() == list.Filtering {
			var cmd tea.Cmd
			m.list, cmd = m.list.Update(msg)
			return m, cmd
		}

		// Handle 'dd' for delete (vim-style)
		if msg.String() == "d" {
			if m.pendingDelete {
				m.deleteCurrentReminder()
				m.pendingDelete = false
			} else {
				m.pendingDelete = true
			}
			return m, nil
		}
		m.pendingDelete = false

		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit

		case key.Matches(msg, keys.Acknowledge):
			r := m.selectedReminder()
			if r != nil && (r.Status == reminder.Pending || r.Status == reminder.Triggered) {
				r.Status = reminder.Acknowledged
				m.refreshList()
			}
			return m, nil

		case key.Matches(msg, keys.Unacknowledge):
			r := m.selectedReminder()
			if r != nil && r.Status == reminder.Acknowledged {
				// Restore to appropriate state based on whether time has passed
				if time.Now().After(r.DateTime) {
					r.Status = reminder.Triggered
				} else {
					r.Status = reminder.Pending
				}
				m.refreshList()
			}
			return m, nil

		case key.Matches(msg, keys.Snooze5m):
			m.snooze(5 * time.Minute)
			return m, nil

		case key.Matches(msg, keys.Snooze1h):
			m.snooze(1 * time.Hour)
			return m, nil

		case key.Matches(msg, keys.Snooze1d):
			m.snooze(24 * time.Hour)
			return m, nil
		}

	case TickMsg:
		// Check for newly triggered reminders
		now := time.Now()
		changed := false
		for _, r := range m.reminders {
			if r.Status == reminder.Pending && now.After(r.DateTime) {
				r.Status = reminder.Triggered
				changed = true
			}
		}
		if changed {
			m.refreshList()
		}
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width-4, msg.Height-6)

	case FileUpdateMsg:
		m.reminders = reminder.MergeFromFile(m.reminders, msg.FilePath, msg.Reminders)
		reminder.SortByDateTime(m.reminders)
		m.refreshList()
		return m, m.waitForFileUpdate()
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(m.list.View())
	b.WriteString("\n")
	b.WriteString(statusBarStyle.Render("enter: done  u: unack  1/2/3: snooze 5m/1h/1d  dd: delete  /: filter  q: quit"))

	return appStyle.Render(b.String())
}

// Key bindings
type keyMap struct {
	Acknowledge   key.Binding
	Unacknowledge key.Binding
	Snooze5m      key.Binding
	Snooze1h      key.Binding
	Snooze1d      key.Binding
	Quit          key.Binding
}

var keys = keyMap{
	Acknowledge: key.NewBinding(
		key.WithKeys("enter", " "),
		key.WithHelp("enter/space", "acknowledge"),
	),
	Unacknowledge: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "unacknowledge"),
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
