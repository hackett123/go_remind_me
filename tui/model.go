package tui

import (
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"go_remind/reminder"
	"go_remind/state"
)

// Input modes
type inputMode int

const (
	modeNormal inputMode = iota
	modeFilter
	modeAdd
	modeTheme
	modeDetail
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
	list          list.Model
	reminders     []*reminder.Reminder
	watcherEvents <-chan FileUpdateMsg
	store         *state.Store
	pendingDelete bool
	pendingG      bool
	width         int
	height        int

	// Grid mode
	gridIndex   int
	gridColumns int
	gridScroll  int // row offset for grid scrolling

	// Compact mode
	compactIndex  int
	compactScroll int // line offset for compact scrolling

	// Sorting
	sortEnabled bool

	// Input handling
	mode            inputMode
	filterInput     textinput.Model
	addInput        textinput.Model
	inputError      string
	editingReminder *reminder.Reminder // non-nil when editing an existing reminder

	// Theme picker
	themeIndex    int
	previewTheme  int
	originalTheme int

	// Detail view
	detailReminder *reminder.Reminder
	detailScroll   int

	// Help
	help help.Model
	keys keyMap

	// Status message (shown after actions)
	statusMessage     string
	statusMessageTime time.Time
}

// New creates a new TUI model with the given reminders
func New(reminders []*reminder.Reminder, watcherEvents <-chan FileUpdateMsg, store *state.Store) Model {
	// Apply default theme
	themes[0].applyStyles()

	items := remindersToItems(reminders)

	l := list.New(items, itemDelegate{}, 80, 20)
	l.Title = ""
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
		store:         store,
		mode:          modeNormal,
		filterInput:   fi,
		addInput:      ai,
		help:          h,
		keys:          keys,
		sortEnabled:   true,
	}
}

// Init initializes the model and starts the tick timer
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		tickCmd(),
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
