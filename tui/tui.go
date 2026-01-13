package tui

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"go_remind/datetime"
	"go_remind/reminder"
)

// Input modes
type inputMode int

const (
	modeNormal inputMode = iota
	modeFilter
	modeAdd
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

	// Input box styles
	inputBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)

	inputLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true)

	inputHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)
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

	// Input handling
	mode        inputMode
	filterInput textinput.Model
	addInput    textinput.Model
	inputError  string

	// Help
	help help.Model
	keys keyMap
}

// New creates a new TUI model with the given reminders
func New(reminders []*reminder.Reminder, watcherEvents <-chan FileUpdateMsg) Model {
	items := remindersToItems(reminders)

	l := list.New(items, itemDelegate{}, 80, 20)
	l.Title = "Go Remind"
	l.Styles.Title = titleStyle
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false) // We'll handle filtering ourselves
	l.SetShowHelp(false)

	// Filter input
	fi := textinput.New()
	fi.Placeholder = "type to filter..."
	fi.CharLimit = 100
	fi.Width = 40

	// Add reminder input
	ai := textinput.New()
	ai.Placeholder = "+1h Call mom  or  Jan 15 2:30pm Meeting"
	ai.CharLimit = 200
	ai.Width = 50

	h := help.New()

	return Model{
		list:          l,
		reminders:     reminders,
		watcherEvents: watcherEvents,
		mode:          modeNormal,
		filterInput:   fi,
		addInput:      ai,
		help:          h,
		keys:          keys,
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

// refreshList updates the list items from the current reminders, applying filter if active
func (m *Model) refreshList() {
	var filtered []*reminder.Reminder
	filterText := strings.ToLower(m.filterInput.Value())

	if filterText == "" {
		filtered = m.reminders
	} else {
		for _, r := range m.reminders {
			if strings.Contains(strings.ToLower(r.Description), filterText) {
				filtered = append(filtered, r)
			}
		}
	}

	items := remindersToItems(filtered)
	m.list.SetItems(items)
}

// selectedReminder returns the currently selected reminder, or nil if none
func (m *Model) selectedReminder() *reminder.Reminder {
	item := m.list.SelectedItem()
	if item == nil {
		return nil
	}
	ri, ok := item.(reminderItem)
	if !ok {
		return nil
	}
	return ri.reminder
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
	r := m.selectedReminder()
	if r == nil {
		return
	}
	// Find and remove
	for i, rem := range m.reminders {
		if rem == r {
			m.reminders = append(m.reminders[:i], m.reminders[i+1:]...)
			break
		}
	}
	m.refreshList()
}

// addReminder parses the input and adds a new reminder
func (m *Model) addReminder(input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("empty input")
	}

	// Parse: first try to find a datetime, rest is description
	words := strings.Fields(input)
	if len(words) < 2 {
		return fmt.Errorf("need both time and description (e.g., '+1h Call mom')")
	}

	now := time.Now()

	// Try parsing increasing numbers of words as the datetime
	for numDateWords := 1; numDateWords < len(words); numDateWords++ {
		dateStr := strings.Join(words[:numDateWords], " ")
		descStr := strings.Join(words[numDateWords:], " ")

		parsedTime, err := datetime.Parse(dateStr, now)
		if err == nil {
			r := &reminder.Reminder{
				DateTime:    parsedTime,
				Description: descStr,
				SourceFile:  "(added in TUI)",
				Status:      reminder.Pending,
			}
			m.reminders = append(m.reminders, r)
			reminder.SortByDateTime(m.reminders)
			m.refreshList()
			return nil
		}
	}

	return fmt.Errorf("couldn't parse time from input")
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle based on mode
		switch m.mode {
		case modeFilter:
			return m.updateFilterMode(msg)
		case modeAdd:
			return m.updateAddMode(msg)
		default:
			return m.updateNormalMode(msg)
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
		m.help.Width = msg.Width
		listHeight := msg.Height - 10 // Leave room for input boxes
		if listHeight < 5 {
			listHeight = 5
		}
		m.list.SetSize(msg.Width-4, listHeight)

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

func (m Model) updateNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

	case key.Matches(msg, keys.Filter):
		m.mode = modeFilter
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, keys.Add):
		m.mode = modeAdd
		m.addInput.Reset()
		m.addInput.Focus()
		m.inputError = ""
		return m, textinput.Blink

	case key.Matches(msg, keys.Help):
		m.help.ShowAll = !m.help.ShowAll
		return m, nil

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

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) updateFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.mode = modeNormal
		m.filterInput.Blur()
		m.filterInput.Reset()
		m.refreshList()
		return m, nil
	case tea.KeyEnter:
		m.mode = modeNormal
		m.filterInput.Blur()
		// Keep the filter applied
		return m, nil
	}

	var cmd tea.Cmd
	m.filterInput, cmd = m.filterInput.Update(msg)
	m.refreshList()
	return m, cmd
}

func (m Model) updateAddMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		m.mode = modeNormal
		m.addInput.Blur()
		m.addInput.Reset()
		m.inputError = ""
		return m, nil
	case tea.KeyEnter:
		err := m.addReminder(m.addInput.Value())
		if err != nil {
			m.inputError = err.Error()
			return m, nil
		}
		m.mode = modeNormal
		m.addInput.Blur()
		m.addInput.Reset()
		m.inputError = ""
		return m, nil
	}

	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	b.WriteString(m.list.View())

	// Show input boxes based on mode
	switch m.mode {
	case modeFilter:
		label := inputLabelStyle.Render("🔍 Filter: ")
		input := m.filterInput.View()
		hint := inputHintStyle.Render("  (enter to apply, esc to cancel)")
		box := inputBoxStyle.Render(label + input + hint)
		b.WriteString("\n")
		b.WriteString(box)

	case modeAdd:
		label := inputLabelStyle.Render("➕ New Reminder: ")
		input := m.addInput.View()
		box := inputBoxStyle.Render(label + input)
		b.WriteString("\n")
		b.WriteString(box)

		hint := inputHintStyle.Render("  Format: <time> <description>  •  Examples: +1h Call mom  |  Jan 15 2:30pm Meeting")
		b.WriteString("\n")
		b.WriteString(hint)

		if m.inputError != "" {
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			b.WriteString("\n")
			b.WriteString(errStyle.Render("  ⚠ " + m.inputError))
		}

	default:
		// Show filter indicator if filter is active
		if m.filterInput.Value() != "" {
			filterIndicator := inputLabelStyle.Render(fmt.Sprintf("🔍 Filtered: %q", m.filterInput.Value()))
			clearHint := inputHintStyle.Render("  (/ to modify, esc in filter to clear)")
			b.WriteString("\n")
			b.WriteString(filterIndicator + clearHint)
		}

		b.WriteString("\n")
		b.WriteString(m.help.View(m.keys))
	}

	return appStyle.Render(b.String())
}

// Key bindings
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
		{k.Filter, k.Add, k.Help, k.Quit},
	}
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
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
}
