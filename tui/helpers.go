package tui

import (
	"fmt"
	"strings"
	"time"

	"go_remind/datetime"
	"go_remind/reminder"
)

// saveState persists the current reminders to disk
func (m *Model) saveState() {
	if m.store == nil {
		return
	}
	// Save in background to avoid blocking UI
	go func() {
		_ = m.store.Save(m.reminders) // Ignore errors for now
	}()
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
	items := m.getFilteredReminders()
	
	if currentLayout == LayoutCard {
		if m.gridIndex >= 0 && m.gridIndex < len(items) {
			return items[m.gridIndex]
		}
		return nil
	}
	
	// Compact mode with sorting
	if m.sortEnabled {
		if m.compactIndex >= 0 && m.compactIndex < len(items) {
			return items[m.compactIndex]
		}
		return nil
	}
	
	// Compact mode without sorting - use list
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
	m.saveState()
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
	m.saveState()
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

	// Try parsing from longest to shortest datetime prefix
	for numDateWords := len(words) - 1; numDateWords >= 1; numDateWords-- {
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
			m.saveState()
			return nil
		}
	}

	return fmt.Errorf("couldn't parse time from input")
}

// updateReminder parses the input and updates an existing reminder
func (m *Model) updateReminder(r *reminder.Reminder, input string) error {
	input = strings.TrimSpace(input)
	if input == "" {
		return fmt.Errorf("empty input")
	}

	words := strings.Fields(input)
	if len(words) < 2 {
		return fmt.Errorf("need both time and description (e.g., '+1h Call mom')")
	}

	now := time.Now()

	// Try parsing from longest to shortest datetime prefix
	for numDateWords := len(words) - 1; numDateWords >= 1; numDateWords-- {
		dateStr := strings.Join(words[:numDateWords], " ")
		descStr := strings.Join(words[numDateWords:], " ")

		parsedTime, err := datetime.Parse(dateStr, now)
		if err == nil {
			r.DateTime = parsedTime
			r.Description = descStr
			// Update status based on new time
			if now.After(parsedTime) {
				if r.Status != reminder.Acknowledged {
					r.Status = reminder.Triggered
				}
			} else {
				if r.Status == reminder.Triggered {
					r.Status = reminder.Pending
				}
			}
			reminder.SortByDateTime(m.reminders)
			m.refreshList()
			m.saveState()
			return nil
		}
	}

	return fmt.Errorf("couldn't parse time from input")
}

func (m Model) getFilteredReminders() []*reminder.Reminder {
	filterText := strings.ToLower(m.filterInput.Value())
	if filterText == "" {
		return m.reminders
	}
	var filtered []*reminder.Reminder
	for _, r := range m.reminders {
		if strings.Contains(strings.ToLower(r.Description), filterText) {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

// scrollToSelection adjusts scroll offset to ensure selected item is visible
func (m *Model) scrollToSelection() {
	if currentLayout == LayoutCard {
		m.scrollGridToSelection()
	} else if m.sortEnabled {
		m.scrollCompactToSelection()
	}
}

// scrollGridToSelection ensures the selected card's row is visible
func (m *Model) scrollGridToSelection() {
	if m.gridColumns < 1 {
		return
	}
	selectedRow := m.gridIndex / m.gridColumns
	visibleRows := m.visibleGridRows()

	// Scroll up if selection is above visible area
	if selectedRow < m.gridScroll {
		m.gridScroll = selectedRow
	}

	// Scroll down if selection is below visible area
	if selectedRow >= m.gridScroll+visibleRows {
		m.gridScroll = selectedRow - visibleRows + 1
	}

	if m.gridScroll < 0 {
		m.gridScroll = 0
	}
}

// scrollCompactToSelection ensures the selected item is visible
func (m *Model) scrollCompactToSelection() {
	visibleItems := m.visibleCompactItems()

	// Scroll up if selection is above visible area
	if m.compactIndex < m.compactScroll {
		m.compactScroll = m.compactIndex
	}

	// Scroll down if selection is below visible area
	if m.compactIndex >= m.compactScroll+visibleItems {
		m.compactScroll = m.compactIndex - visibleItems + 1
	}

	if m.compactScroll < 0 {
		m.compactScroll = 0
	}
}

// visibleGridRows returns how many card rows fit in the available height
func (m *Model) visibleGridRows() int {
	// Card height: 4 content + 2 border + 1 margin = 7 lines per row
	cardRowHeight := 7
	availableHeight := m.height - 4 // leave room for help bar and scroll indicators
	if availableHeight < cardRowHeight {
		return 1
	}
	return availableHeight / cardRowHeight
}

// visibleCompactItems returns how many items fit in the available height
// Each item is 1 line, plus we account for ~3 section headers
func (m *Model) visibleCompactItems() int {
	availableHeight := m.height - 4 // leave room for help bar and scroll indicators
	availableHeight -= 3            // approximate space for section headers
	if availableHeight < 1 {
		return 1
	}
	return availableHeight
}
