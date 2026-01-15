package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"go_remind/reminder"
)

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
		case modeTheme:
			return m.updateThemeMode(msg)
		case modeDetail:
			return m.updateDetailMode(msg)
		default:
			return m.updateNormalMode(msg)
		}

	case TickMsg:
		// Check for newly triggered reminders
		changed := false
		for _, r := range m.reminders {
			if r.Status == reminder.Pending && r.IsDue() {
				r.Status = reminder.Triggered
				changed = true
			}
		}
		if changed {
			m.refreshList()
			m.saveState()
		}
		// Clear status message after 3 seconds
		if m.statusMessage != "" && time.Since(m.statusMessageTime) > 3*time.Second {
			m.statusMessage = ""
		}
		return m, tickCmd()

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.Width = msg.Width
		listHeight := msg.Height - 4
		if listHeight < 5 {
			listHeight = 5
		}
		m.list.SetSize(msg.Width-4, listHeight)
		// Calculate grid columns (card width ~40 + margin)
		m.gridColumns = (msg.Width - 4) / 40
		if m.gridColumns < 1 {
			m.gridColumns = 1
		}

	case FileUpdateMsg:
		m.reminders = reminder.MergeFromFile(m.reminders, msg.FilePath, msg.Reminders)
		reminder.SortByDateTime(m.reminders)
		m.refreshList()
		m.saveState()
		m.setStatusMessage(fmt.Sprintf("File updated: %d reminders", len(msg.Reminders)))
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
			r := m.selectedReminder()
			if r != nil {
				desc := r.Description
				m.deleteCurrentReminder()
				m.setStatusMessage("Deleted: " + desc)
			}
			m.pendingDelete = false
		} else {
			m.pendingDelete = true
		}
		m.pendingG = false
		return m, nil
	}
	m.pendingDelete = false

	// Handle 'gg' for go to first (vim-style)
	if msg.String() == "g" {
		if m.pendingG {
			// gg - go to first item
			m.gotoFirstItem()
			m.pendingG = false
		} else {
			m.pendingG = true
		}
		return m, nil
	}
	m.pendingG = false

	// Handle 'G' for go to last
	if msg.String() == "G" {
		m.gotoLastItem()
		return m, nil
	}

	// Handle '{' and '}' for section navigation
	if msg.String() == "{" {
		m.gotoPrevSection()
		return m, nil
	}
	if msg.String() == "}" {
		m.gotoNextSection()
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Theme):
		m.mode = modeTheme
		m.originalTheme = m.themeIndex
		m.previewTheme = m.themeIndex
		return m, nil

	case key.Matches(msg, keys.Layout):
		currentLayout = (currentLayout + 1) % LayoutMode(len(layoutNames))
		m.list.SetDelegate(itemDelegate{})
		return m, nil

	case key.Matches(msg, keys.Sort):
		m.sortEnabled = !m.sortEnabled
		return m, nil

	case key.Matches(msg, keys.Filter):
		m.mode = modeFilter
		m.filterInput.Focus()
		return m, textinput.Blink

	case key.Matches(msg, keys.Add):
		m.mode = modeAdd
		m.addInput.Reset()
		m.addInput.Focus()
		m.inputError = ""
		m.editingReminder = nil
		return m, textinput.Blink

	case key.Matches(msg, keys.Edit):
		r := m.selectedReminder()
		if r == nil {
			return m, nil
		}
		m.mode = modeAdd
		m.editingReminder = r
		// Format: yyyy-mm-dd hh:mm description
		prefill := r.DateTime.Format("2006-01-02 15:04") + " " + r.Description
		m.addInput.SetValue(prefill)
		m.addInput.Focus()
		m.addInput.CursorEnd()
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
			m.saveState()
			m.setStatusMessage("Acknowledged: " + r.Description)
		}
		return m, nil

	case key.Matches(msg, keys.Unacknowledge):
		r := m.selectedReminder()
		if r != nil && r.Status == reminder.Acknowledged {
			if r.IsDue() {
				r.Status = reminder.Triggered
			} else {
				r.Status = reminder.Pending
			}
			m.refreshList()
			m.saveState()
			m.setStatusMessage("Unacknowledged: " + r.Description)
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

	case msg.String() == "K":
		r := m.selectedReminder()
		if r != nil {
			m.mode = modeDetail
			m.detailReminder = r
			m.detailScroll = 0
		}
		return m, nil
	}

	// Handle navigation in card mode
	if currentLayout == LayoutCard {
		items := m.getFilteredReminders()
		maxIdx := len(items) - 1
		if maxIdx < 0 {
			maxIdx = 0
		}
		switch {
		case key.Matches(msg, keys.Up):
			m.gridIndex -= m.gridColumns
			if m.gridIndex < 0 {
				m.gridIndex = 0
			}
			m.scrollToSelection()
			return m, nil
		case key.Matches(msg, keys.Down):
			m.gridIndex += m.gridColumns
			if m.gridIndex > maxIdx {
				m.gridIndex = maxIdx
			}
			m.scrollToSelection()
			return m, nil
		case msg.String() == "h" || msg.String() == "left":
			if m.gridIndex > 0 {
				m.gridIndex--
			}
			m.scrollToSelection()
			return m, nil
		case msg.String() == "l" || msg.String() == "right":
			if m.gridIndex < maxIdx {
				m.gridIndex++
			}
			m.scrollToSelection()
			return m, nil
		}
	}

	// Handle navigation in compact mode with sorting
	if currentLayout == LayoutCompact && m.sortEnabled {
		items := m.getFilteredReminders()
		maxIdx := len(items) - 1
		if maxIdx < 0 {
			maxIdx = 0
		}
		switch {
		case key.Matches(msg, keys.Up):
			if m.compactIndex > 0 {
				m.compactIndex--
			}
			m.scrollToSelection()
			return m, nil
		case key.Matches(msg, keys.Down):
			if m.compactIndex < maxIdx {
				m.compactIndex++
			}
			m.scrollToSelection()
			return m, nil
		}
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
		m.editingReminder = nil
		return m, nil
	case tea.KeyEnter:
		var err error
		if m.editingReminder != nil {
			err = m.updateReminder(m.editingReminder, m.addInput.Value())
		} else {
			err = m.addReminder(m.addInput.Value())
		}
		if err != nil {
			m.inputError = err.Error()
			return m, nil
		}
		m.mode = modeNormal
		m.addInput.Blur()
		m.addInput.Reset()
		m.inputError = ""
		m.editingReminder = nil
		return m, nil
	}

	var cmd tea.Cmd
	m.addInput, cmd = m.addInput.Update(msg)
	return m, cmd
}

func (m Model) updateThemeMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEscape:
		// Restore original theme
		m.themeIndex = m.originalTheme
		themes[m.themeIndex].applyStyles()
		m.mode = modeNormal
		return m, nil
	case tea.KeyEnter:
		// Confirm selection
		m.themeIndex = m.previewTheme
		m.mode = modeNormal
		return m, nil
	case tea.KeyUp, tea.KeyShiftTab:
		if m.previewTheme > 0 {
			m.previewTheme--
			themes[m.previewTheme].applyStyles()
		}
		return m, nil
	case tea.KeyDown, tea.KeyTab:
		if m.previewTheme < len(themes)-1 {
			m.previewTheme++
			themes[m.previewTheme].applyStyles()
		}
		return m, nil
	}

	switch msg.String() {
	case "k":
		if m.previewTheme > 0 {
			m.previewTheme--
			themes[m.previewTheme].applyStyles()
		}
	case "j":
		if m.previewTheme < len(themes)-1 {
			m.previewTheme++
			themes[m.previewTheme].applyStyles()
		}
	}
	return m, nil
}

func (m Model) updateDetailMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle 'dd' for delete
	if msg.String() == "d" {
		if m.pendingDelete {
			if m.detailReminder != nil {
				desc := m.detailReminder.Description
				m.deleteCurrentReminder()
				m.setStatusMessage("Deleted: " + desc)
			}
			m.pendingDelete = false
			m.mode = modeNormal
			m.detailReminder = nil
			m.detailScroll = 0
			return m, nil
		}
		m.pendingDelete = true
		return m, nil
	}
	m.pendingDelete = false

	switch msg.Type {
	case tea.KeyEscape:
		m.mode = modeNormal
		m.detailReminder = nil
		m.detailScroll = 0
		return m, nil
	case tea.KeyUp:
		if m.detailScroll > 0 {
			m.detailScroll--
		}
		return m, nil
	case tea.KeyDown:
		m.detailScroll++
		return m, nil
	case tea.KeyEnter, tea.KeySpace:
		if m.detailReminder != nil && (m.detailReminder.Status == reminder.Pending || m.detailReminder.Status == reminder.Triggered) {
			m.detailReminder.Status = reminder.Acknowledged
			m.refreshList()
			m.saveState()
			m.setStatusMessage("Acknowledged: " + m.detailReminder.Description)
		}
		return m, nil
	}

	switch msg.String() {
	case "k":
		if m.detailScroll > 0 {
			m.detailScroll--
		}
	case "j":
		m.detailScroll++
	case "u":
		if m.detailReminder != nil && m.detailReminder.Status == reminder.Acknowledged {
			if m.detailReminder.IsDue() {
				m.detailReminder.Status = reminder.Triggered
			} else {
				m.detailReminder.Status = reminder.Pending
			}
			m.refreshList()
			m.saveState()
			m.setStatusMessage("Unacknowledged: " + m.detailReminder.Description)
		}
	case "1":
		if m.detailReminder != nil && m.detailReminder.Snoozeable() {
			m.detailReminder.DateTime = m.detailReminder.DateTime.Add(5 * time.Minute)
			m.detailReminder.Status = reminder.Pending
			reminder.SortByDateTime(m.reminders)
			m.refreshList()
			m.saveState()
			m.setStatusMessage("Snoozed 5 minutes: " + m.detailReminder.Description)
		}
	case "2":
		if m.detailReminder != nil && m.detailReminder.Snoozeable() {
			m.detailReminder.DateTime = m.detailReminder.DateTime.Add(1 * time.Hour)
			m.detailReminder.Status = reminder.Pending
			reminder.SortByDateTime(m.reminders)
			m.refreshList()
			m.saveState()
			m.setStatusMessage("Snoozed 1 hour: " + m.detailReminder.Description)
		}
	case "3":
		if m.detailReminder != nil && m.detailReminder.Snoozeable() {
			m.detailReminder.DateTime = m.detailReminder.DateTime.Add(24 * time.Hour)
			m.detailReminder.Status = reminder.Pending
			reminder.SortByDateTime(m.reminders)
			m.refreshList()
			m.saveState()
			m.setStatusMessage("Snoozed 1 day: " + m.detailReminder.Description)
		}
	case "e":
		if m.detailReminder != nil {
			m.mode = modeAdd
			m.editingReminder = m.detailReminder
			prefill := m.detailReminder.DateTime.Format("2006-01-02 15:04") + " " + m.detailReminder.Description
			m.addInput.SetValue(prefill)
			m.addInput.Focus()
			m.addInput.CursorEnd()
			m.inputError = ""
			m.detailReminder = nil
			m.detailScroll = 0
			return m, textinput.Blink
		}
	}
	return m, nil
}
