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

func (m Model) compactView() string {
	items := m.getFilteredReminders()
	if len(items) == 0 {
		return normalStyle.Render("No reminders")
	}

	if !m.sortEnabled {
		// No sorting - use regular list view
		return m.list.View()
	}

	// Sort into sections
	now := time.Now()
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location())
	tomorrowEnd := todayEnd.Add(24 * time.Hour)

	var due, comingUp, tomorrow []*reminder.Reminder
	for _, r := range items {
		if r.DateTime.Before(now) {
			due = append(due, r)
		} else if r.DateTime.Before(todayEnd) {
			comingUp = append(comingUp, r)
		} else if r.DateTime.Before(tomorrowEnd) {
			tomorrow = append(tomorrow, r)
		} else {
			tomorrow = append(tomorrow, r)
		}
	}

	sectionStyle := lipgloss.NewStyle().
		Foreground(titleStyle.GetForeground()).
		Bold(true).
		MarginTop(1).
		MarginBottom(0)

	var sections []string

	if len(due) > 0 {
		sections = append(sections, sectionStyle.Render("Due"))
		sections = append(sections, m.renderCompactSection(due))
	}

	if len(comingUp) > 0 {
		sections = append(sections, sectionStyle.Render("Coming Up!"))
		sections = append(sections, m.renderCompactSection(comingUp))
	}

	if len(tomorrow) > 0 {
		sections = append(sections, sectionStyle.Render("Tomorrow and beyond..."))
		sections = append(sections, m.renderCompactSection(tomorrow))
	}

	if len(sections) == 0 {
		return normalStyle.Render("No pending reminders")
	}

	return strings.Join(sections, "\n")
}

func (m Model) renderCompactSection(items []*reminder.Reminder) string {
	var lines []string
	globalIdx := 0
	allItems := m.getFilteredReminders()
	
	for _, r := range items {
		// Find this reminder's global index
		for i, ar := range allItems {
			if ar == r {
				globalIdx = i
				break
			}
		}
		
		timeStr := r.DateTime.Format("Jan 2 3:04pm")
		source := r.SourceFile
		if source == "" {
			source = "(added in TUI)"
		}

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
	return strings.Join(lines, "\n")
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
		b.WriteString(m.gridView())
	} else {
		b.WriteString(m.compactView())
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
