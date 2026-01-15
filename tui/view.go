package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"go_remind/reminder"
)

// welcomeView renders the welcome screen for standalone mode
func (m Model) welcomeView() string {
	width := m.width
	if width == 0 {
		width = 80
	}

	var lines []string

	lines = append(lines, welcomeTitleStyle.Render("Welcome to Go Remind Me!"))
	lines = append(lines, "")
	lines = append(lines, welcomeTextStyle.Render("A simple terminal reminder app."))
	lines = append(lines, "")
	lines = append(lines, welcomeTextStyle.Render("Get started:"))
	lines = append(lines, welcomeTextStyle.Render("Press ")+welcomeHighlightStyle.Render("n")+welcomeTextStyle.Render(" to add a new reminder"))
	lines = append(lines, welcomeTextStyle.Render("Press ")+welcomeHighlightStyle.Render("?")+welcomeTextStyle.Render(" to see all commands"))
	lines = append(lines, "")
	lines = append(lines, welcomeTextStyle.Render("Or run with a file/directory:"))
	lines = append(lines, inputHintStyle.Render("go_remind notes.md"))
	lines = append(lines, inputHintStyle.Render("go_remind ~/notes/"))
	lines = append(lines, "")
	lines = append(lines, welcomeTextStyle.Render("Reminders in markdown use:"))
	lines = append(lines, inputHintStyle.Render("[remind_me 3pm Call mom]"))
	lines = append(lines, inputHintStyle.Render("[remind_me +1h Check oven]"))

	// Center each line
	var centered []string
	for _, line := range lines {
		centered = append(centered, lipgloss.PlaceHorizontal(width-4, lipgloss.Center, line))
	}

	return strings.Join(centered, "\n")
}

func (m Model) compactViewContent() string {
	items := m.getFilteredReminders()
	if len(items) == 0 {
		return normalStyle.Render("No reminders")
	}

	// Sort into sections
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

	var due, comingUp, tomorrow, laterThisWeek, nextWeek, laterThisMonth, beyondNextMonth []*reminder.Reminder
	for _, r := range items {
		if r.DateTime.Before(now) {
			due = append(due, r)
		} else if r.DateTime.Before(todayEnd) {
			comingUp = append(comingUp, r)
		} else if r.DateTime.Before(tomorrowEnd) {
			tomorrow = append(tomorrow, r)
		} else if r.DateTime.Before(thisWeekEnd) {
			laterThisWeek = append(laterThisWeek, r)
		} else if r.DateTime.Before(nextWeekEnd) {
			nextWeek = append(nextWeek, r)
		} else if r.DateTime.Before(thisMonthEnd) {
			laterThisMonth = append(laterThisMonth, r)
		} else if r.DateTime.Before(nextMonthEnd) {
			beyondNextMonth = append(beyondNextMonth, r)
		} else {
			beyondNextMonth = append(beyondNextMonth, r)
		}
	}

	sectionStyle := lipgloss.NewStyle().
		Foreground(titleStyle.GetForeground()).
		Bold(true).
		MarginTop(1).
		MarginBottom(0)

	// Determine visible item range
	visibleItems := m.visibleCompactItems()
	totalItems := len(items)
	startItem := m.compactScroll
	endItem := m.compactScroll + visibleItems
	if endItem > totalItems {
		endItem = totalItems
	}

	var output []string

	// Scroll up indicator
	if m.compactScroll > 0 {
		output = append(output, sourceStyle.Render(fmt.Sprintf("  ↑ %d more items above", m.compactScroll)))
	}

	// Render only items in visible range, with section headers
	itemIdx := 0
	addSection := func(items []*reminder.Reminder, title string) {
		if len(items) > 0 {
			sectionStart := itemIdx
			sectionEnd := itemIdx + len(items)
			if sectionEnd > startItem && sectionStart < endItem {
				output = append(output, sectionStyle.Render(title))
				output = append(output, m.renderCompactLinesInRange(items, sectionStart, startItem, endItem)...)
			}
			itemIdx = sectionEnd
		}
	}

	addSection(due, "Due")
	addSection(comingUp, "Coming Up!")
	addSection(tomorrow, "Tomorrow")
	addSection(laterThisWeek, "Later This Week")
	addSection(nextWeek, "Next Week")
	addSection(laterThisMonth, "Later This Month")
	addSection(beyondNextMonth, "Next Month & Beyond")

	// Scroll down indicator
	if endItem < totalItems {
		output = append(output, sourceStyle.Render(fmt.Sprintf("  ↓ %d more items below", totalItems-endItem)))
	}

	if len(output) == 0 {
		return normalStyle.Render("No pending reminders")
	}

	return strings.Join(output, "\n")
}

// renderCompactLinesInRange renders items from a section that fall within the visible range
// sectionStart is the global index of the first item in this section
// startItem/endItem define the visible range
func (m Model) renderCompactLinesInRange(items []*reminder.Reminder, sectionStart, startItem, endItem int) []string {
	var lines []string

	for i, r := range items {
		globalIdx := sectionStart + i

		// Skip items outside visible range
		if globalIdx < startItem || globalIdx >= endItem {
			continue
		}

		timeStr := r.DateTime.Format("Jan 2 3:04pm")

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

		// Highlight selected item
		if globalIdx == m.compactIndex {
			statusIcon = "▸"
			if r.Status != reminder.Triggered && r.Status != reminder.Acknowledged {
				style = selectedItemStyle
			}
		}

		line := fmt.Sprintf("%s %-18s %-12s %s", statusIcon, timeStr, r.Status.String(), r.Description)
		lines = append(lines, style.Render(line))
	}
	return lines
}

func (m Model) themePickerView() string {
	var b strings.Builder
	b.WriteString(inputLabelStyle.Render("🎨 Select Theme"))
	b.WriteString(inputHintStyle.Render("  (↑/k ↓/j to preview, enter to select, esc to cancel)"))
	b.WriteString("\n\n")

	for i, t := range themes {
		cursor := "  "
		if i == m.previewTheme {
			cursor = "▸ "
		}
		name := t.Name
		if i == m.previewTheme {
			name = selectedItemStyle.Render(name)
		} else {
			name = normalStyle.Render(name)
		}
		b.WriteString(cursor + name + "\n")
	}
	return b.String()
}

// View renders the UI
func (m Model) View() string {
	var b strings.Builder

	// Show welcome screen if no reminders and in standalone mode
	if len(m.reminders) == 0 && m.watcherEvents == nil && m.mode == modeNormal {
		b.WriteString(m.welcomeView())
		b.WriteString("\n\n")
		b.WriteString(m.help.View(m.keys))
		return appStyle.Render(b.String())
	}

	// Use grid view for card layout, list view for compact
	if currentLayout == LayoutCard {
		if len(m.reminders) == 0 {
			asciiTitle := `   ___                       _           _   __  __      _
  / __|___    _ _ ___ _ __ (_)_ _  __| | |  \/  |___ | |
 | (_ / _ \  | '_/ -_) '  \| | ' \/ _' | | |\/| / -_)|_|
  \___\___/  |_| \___|_|_|_|_|_||_\__,_| |_|  |_\___/(_)`
			b.WriteString(titleStyle.Render(asciiTitle))
			b.WriteString("\n\n")
		}
		b.WriteString(m.gridViewContent())
	} else if m.sortEnabled {
		b.WriteString(m.compactViewContent())
	} else {
		// Unsorted compact uses built-in list scrolling
		b.WriteString(m.list.View())
	}

	// Show input boxes based on mode
	switch m.mode {
	case modeDetail:
		return appStyle.Render(m.detailView())

	case modeFilter:
		label := inputLabelStyle.Render("🔍 Filter: ")
		input := m.filterInput.View()
		hint := inputHintStyle.Render("  (enter to apply, esc to cancel)")
		box := inputBoxStyle.Render(label + input + hint)
		b.WriteString("\n")
		b.WriteString(box)

		// Show matching tags when typing a tag filter
		filterText := m.filterInput.Value()
		if strings.HasPrefix(filterText, "#") && len(filterText) > 1 {
			tagPrefix := strings.TrimPrefix(filterText, "#")
			matches := m.getMatchingTags(tagPrefix)
			if len(matches) > 0 {
				var tagStrs []string
				for _, tag := range matches {
					tagStrs = append(tagStrs, "#"+tag)
				}
				b.WriteString("\n")
				b.WriteString(inputHintStyle.Render("  Matching tags: ") + tagStyle.Render(strings.Join(tagStrs, "  ")))
			}
		} else if filterText == "#" {
			// Show all available tags when just "#" is typed
			allTags := m.getAllTags()
			if len(allTags) > 0 {
				var tagStrs []string
				for _, tag := range allTags {
					tagStrs = append(tagStrs, "#"+tag)
				}
				b.WriteString("\n")
				b.WriteString(inputHintStyle.Render("  Available tags: ") + tagStyle.Render(strings.Join(tagStrs, "  ")))
			}
		}

	case modeAdd:
		var label string
		if m.editingReminder != nil {
			label = inputLabelStyle.Render("✏️  Edit Reminder: ")
		} else {
			label = inputLabelStyle.Render("➕ New Reminder: ")
		}
		input := m.addInput.View()
		box := inputBoxStyle.Render(label + input)
		b.WriteString("\n")
		b.WriteString(box)

		hint := inputHintStyle.Render("  Format: <time> <description>  •  Examples: +1h Call mom  |  2025-01-15 14:30 Meeting")
		b.WriteString("\n")
		b.WriteString(hint)

		if m.inputError != "" {
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			b.WriteString("\n")
			b.WriteString(errStyle.Render("  ⚠ " + m.inputError))
		}

	case modeTheme:
		b.WriteString("\n")
		b.WriteString(m.themePickerView())

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
