package tui

import (
	"fmt"
	"strings"
	"time"

	"go_remind/datetime"
	"go_remind/parser"
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
			// Extract tags from description
			cleanDesc, tags := parser.ExtractTags(descStr)
			r := &reminder.Reminder{
				DateTime:    parsedTime,
				Description: cleanDesc,
				Tags:        tags,
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
			// Extract tags from description
			cleanDesc, tags := parser.ExtractTags(descStr)
			r.DateTime = parsedTime
			r.Description = cleanDesc
			r.Tags = tags
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

	// Check if filtering by tag (starts with #)
	if strings.HasPrefix(filterText, "#") {
		tagFilter := strings.TrimPrefix(filterText, "#")
		for _, r := range m.reminders {
			for _, tag := range r.Tags {
				if strings.ToLower(tag) == tagFilter {
					filtered = append(filtered, r)
					break
				}
			}
		}
		return filtered
	}

	// Filter by description
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

	// Calculate actual row accounting for sections if sorting is enabled
	selectedRow := m.calculateGridRow(m.gridIndex)
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

// calculateGridRow returns the actual row number for a given item index,
// accounting for section boundaries when sorting is enabled
func (m *Model) calculateGridRow(itemIndex int) int {
	if !m.sortEnabled {
		// No sections - simple calculation
		return itemIndex / m.gridColumns
	}

	// With sections, we need to calculate based on section membership
	items := m.getFilteredReminders()
	if len(items) == 0 {
		return 0
	}

	now := time.Now()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	tomorrowEnd := todayEnd.Add(24 * time.Hour)

	// Calculate week boundaries (week starts on Sunday)
	daysUntilEndOfWeek := (7 - int(now.Weekday())) % 7
	thisWeekEnd := time.Date(now.Year(), now.Month(), now.Day()+daysUntilEndOfWeek, 23, 59, 59, 0, now.Location())
	nextWeekEnd := thisWeekEnd.Add(7 * 24 * time.Hour)

	// Calculate month boundaries
	thisMonthEnd := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, now.Location())
	nextMonthEnd := time.Date(now.Year(), now.Month()+2, 0, 23, 59, 59, 0, now.Location())

	// Count items in each section
	var dueCount, comingUpCount, tomorrowCount, laterThisWeekCount, nextWeekCount, laterThisMonthCount int
	for _, r := range items {
		if r.DateTime.Before(now) {
			dueCount++
		} else if r.DateTime.Before(todayEnd) {
			comingUpCount++
		} else if r.DateTime.Before(tomorrowEnd) {
			tomorrowCount++
		} else if r.DateTime.Before(thisWeekEnd) {
			laterThisWeekCount++
		} else if r.DateTime.Before(nextWeekEnd) {
			nextWeekCount++
		} else if r.DateTime.Before(thisMonthEnd) {
			laterThisMonthCount++
		} else if r.DateTime.Before(nextMonthEnd) {
			// beyondNextMonth - we don't need to count, it's the last section
		}
	}

	// Calculate rows per section (ceiling division)
	cols := m.gridColumns
	ceilDiv := func(a, b int) int {
		if a == 0 {
			return 0
		}
		return (a + b - 1) / b
	}

	dueRows := ceilDiv(dueCount, cols)
	comingUpRows := ceilDiv(comingUpCount, cols)
	tomorrowRows := ceilDiv(tomorrowCount, cols)
	laterThisWeekRows := ceilDiv(laterThisWeekCount, cols)
	nextWeekRows := ceilDiv(nextWeekCount, cols)
	laterThisMonthRows := ceilDiv(laterThisMonthCount, cols)

	// Calculate cumulative counts and rows
	cumCounts := []int{
		dueCount,
		dueCount + comingUpCount,
		dueCount + comingUpCount + tomorrowCount,
		dueCount + comingUpCount + tomorrowCount + laterThisWeekCount,
		dueCount + comingUpCount + tomorrowCount + laterThisWeekCount + nextWeekCount,
		dueCount + comingUpCount + tomorrowCount + laterThisWeekCount + nextWeekCount + laterThisMonthCount,
	}
	cumRows := []int{
		dueRows,
		dueRows + comingUpRows,
		dueRows + comingUpRows + tomorrowRows,
		dueRows + comingUpRows + tomorrowRows + laterThisWeekRows,
		dueRows + comingUpRows + tomorrowRows + laterThisWeekRows + nextWeekRows,
		dueRows + comingUpRows + tomorrowRows + laterThisWeekRows + nextWeekRows + laterThisMonthRows,
	}

	// Determine which section the item is in and calculate row
	if itemIndex < cumCounts[0] {
		return itemIndex / cols
	}
	for i := 0; i < len(cumCounts)-1; i++ {
		if itemIndex < cumCounts[i+1] {
			indexInSection := itemIndex - cumCounts[i]
			return cumRows[i] + indexInSection/cols
		}
	}
	// Last section (beyond next month)
	indexInSection := itemIndex - cumCounts[len(cumCounts)-1]
	return cumRows[len(cumRows)-1] + indexInSection/cols
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
	availableHeight := m.height - 6 // leave room for help bar and scroll indicators (2 lines)
	if availableHeight < cardRowHeight {
		return 1
	}
	return availableHeight / cardRowHeight
}

// visibleCompactItems returns how many items fit in the available height
// Each item is 1 line, plus we account for ~3 section headers
func (m *Model) visibleCompactItems() int {
	availableHeight := m.height - 6 // leave room for help bar, scroll indicators, and some headers
	if availableHeight < 1 {
		return 1
	}
	return availableHeight
}

// getSectionBoundaries returns the starting index of each non-empty section
func (m *Model) getSectionBoundaries() []int {
	items := m.getFilteredReminders()
	if len(items) == 0 {
		return []int{0}
	}

	now := time.Now()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	tomorrowEnd := todayEnd.Add(24 * time.Hour)
	daysUntilEndOfWeek := (7 - int(now.Weekday())) % 7
	thisWeekEnd := time.Date(now.Year(), now.Month(), now.Day()+daysUntilEndOfWeek, 23, 59, 59, 0, now.Location())
	nextWeekEnd := thisWeekEnd.Add(7 * 24 * time.Hour)
	thisMonthEnd := time.Date(now.Year(), now.Month()+1, 0, 23, 59, 59, 0, now.Location())
	nextMonthEnd := time.Date(now.Year(), now.Month()+2, 0, 23, 59, 59, 0, now.Location())

	// Count items in each section
	var counts [7]int
	for _, r := range items {
		if r.DateTime.Before(now) {
			counts[0]++
		} else if r.DateTime.Before(todayEnd) {
			counts[1]++
		} else if r.DateTime.Before(tomorrowEnd) {
			counts[2]++
		} else if r.DateTime.Before(thisWeekEnd) {
			counts[3]++
		} else if r.DateTime.Before(nextWeekEnd) {
			counts[4]++
		} else if r.DateTime.Before(thisMonthEnd) {
			counts[5]++
		} else if r.DateTime.Before(nextMonthEnd) {
			counts[6]++
		} else {
			counts[6]++
		}
	}

	// Build list of section start indices (only for non-empty sections)
	var boundaries []int
	idx := 0
	for _, count := range counts {
		if count > 0 {
			boundaries = append(boundaries, idx)
			idx += count
		}
	}

	if len(boundaries) == 0 {
		return []int{0}
	}
	return boundaries
}

// getNextSectionStart returns the start index of the next section after currentIdx
func (m *Model) getNextSectionStart(currentIdx int) int {
	boundaries := m.getSectionBoundaries()
	items := m.getFilteredReminders()
	maxIdx := len(items) - 1

	for _, boundary := range boundaries {
		if boundary > currentIdx {
			return boundary
		}
	}
	// Already at or past last section, go to end
	return maxIdx
}

// getPrevSectionStart returns the start index of the previous section before currentIdx
func (m *Model) getPrevSectionStart(currentIdx int) int {
	boundaries := m.getSectionBoundaries()

	// Find the section that contains currentIdx, then go to previous
	prevBoundary := 0
	for _, boundary := range boundaries {
		if boundary >= currentIdx {
			break
		}
		prevBoundary = boundary
	}

	// If we're at the start of a section, go to the previous section
	for _, boundary := range boundaries {
		if boundary == currentIdx && prevBoundary < currentIdx {
			// Find the boundary before prevBoundary
			for i := len(boundaries) - 1; i >= 0; i-- {
				if boundaries[i] < currentIdx {
					return boundaries[i]
				}
			}
		}
	}

	return prevBoundary
}

// gotoFirstItem moves selection to the first item
func (m *Model) gotoFirstItem() {
	if currentLayout == LayoutCard {
		m.gridIndex = 0
		m.gridScroll = 0
	} else if m.sortEnabled {
		m.compactIndex = 0
		m.compactScroll = 0
	} else {
		m.list.Select(0)
	}
}

// gotoLastItem moves selection to the last item
func (m *Model) gotoLastItem() {
	items := m.getFilteredReminders()
	maxIdx := len(items) - 1
	if maxIdx < 0 {
		maxIdx = 0
	}

	if currentLayout == LayoutCard {
		m.gridIndex = maxIdx
		m.scrollToSelection()
	} else if m.sortEnabled {
		m.compactIndex = maxIdx
		m.scrollToSelection()
	} else {
		m.list.Select(maxIdx)
	}
}

// gotoPrevSection moves selection to the start of the previous section
func (m *Model) gotoPrevSection() {
	var currentIdx int
	if currentLayout == LayoutCard {
		currentIdx = m.gridIndex
	} else if m.sortEnabled {
		currentIdx = m.compactIndex
	} else {
		currentIdx = m.list.Index()
	}

	newIdx := m.getPrevSectionStart(currentIdx)

	if currentLayout == LayoutCard {
		m.gridIndex = newIdx
		m.scrollToSelection()
	} else if m.sortEnabled {
		m.compactIndex = newIdx
		m.scrollToSelection()
	} else {
		m.list.Select(newIdx)
	}
}

// gotoNextSection moves selection to the start of the next section
func (m *Model) gotoNextSection() {
	var currentIdx int
	if currentLayout == LayoutCard {
		currentIdx = m.gridIndex
	} else if m.sortEnabled {
		currentIdx = m.compactIndex
	} else {
		currentIdx = m.list.Index()
	}

	newIdx := m.getNextSectionStart(currentIdx)

	if currentLayout == LayoutCard {
		m.gridIndex = newIdx
		m.scrollToSelection()
	} else if m.sortEnabled {
		m.compactIndex = newIdx
		m.scrollToSelection()
	} else {
		m.list.Select(newIdx)
	}
}
