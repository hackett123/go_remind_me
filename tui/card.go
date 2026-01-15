package tui

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"go_remind/reminder"
)

func (m Model) gridView() string {
	items := m.getFilteredReminders()
	if len(items) == 0 {
		return normalStyle.Render("No reminders")
	}

	cardWidth := 38
	cols := m.gridColumns
	if cols < 1 {
		cols = 1
	}

	if !m.sortEnabled {
		// No sorting - render all cards in grid
		var rows []string
		for i := 0; i < len(items); i += cols {
			var rowCards []string
			for j := 0; j < cols && i+j < len(items); j++ {
				idx := i + j
				rowCards = append(rowCards, m.renderCard(items[idx], idx, cardWidth))
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
		}
		return lipgloss.JoinVertical(lipgloss.Left, rows...)
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
		MarginBottom(1)

	var sections []string
	globalIdx := 0

	if len(due) > 0 {
		sections = append(sections, sectionStyle.Render("Due"))
		sections = append(sections, m.renderSection(due, &globalIdx, cols, cardWidth))
	}

	if len(comingUp) > 0 {
		sections = append(sections, sectionStyle.Render("Coming Up!"))
		sections = append(sections, m.renderSection(comingUp, &globalIdx, cols, cardWidth))
	}

	if len(tomorrow) > 0 {
		sections = append(sections, sectionStyle.Render("Tomorrow and beyond..."))
		sections = append(sections, m.renderSection(tomorrow, &globalIdx, cols, cardWidth))
	}

	if len(sections) == 0 {
		return normalStyle.Render("No pending reminders")
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

func (m Model) renderSection(items []*reminder.Reminder, globalIdx *int, cols, cardWidth int) string {
	var rows []string
	for i := 0; i < len(items); i += cols {
		var rowCards []string
		for j := 0; j < cols && i+j < len(items); j++ {
			idx := i + j
			rowCards = append(rowCards, m.renderCard(items[idx], *globalIdx, cardWidth))
			*globalIdx++
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (m Model) renderCard(r *reminder.Reminder, index, width int) string {
	timeStr := r.DateTime.Format("Jan 2 3:04pm")
	source := filepath.Base(r.SourceFile)
	isSelected := index == m.gridIndex

	var style lipgloss.Style
	var borderColor lipgloss.TerminalColor
	switch r.Status {
	case reminder.Triggered:
		style = triggeredStyle
		borderColor = triggeredStyle.GetForeground()
	case reminder.Acknowledged:
		style = acknowledgedStyle
		borderColor = acknowledgedStyle.GetForeground()
	default:
		style = normalStyle
		borderColor = normalStyle.GetForeground()
	}

	if isSelected {
		borderColor = selectedItemStyle.GetForeground()
		if r.Status != reminder.Triggered && r.Status != reminder.Acknowledged {
			style = selectedItemStyle
		}
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(0, 1).
		Width(width).
		Height(4).
		MarginRight(1)

	desc := r.Description
	maxWidth := width - 4

	// Wrap description to two lines at word boundaries
	var line1, line2 string
	if len(desc) <= maxWidth {
		line1 = desc
	} else {
		// Find last space before maxWidth
		breakPoint := maxWidth
		for i := maxWidth; i > 0; i-- {
			if desc[i] == ' ' {
				breakPoint = i
				break
			}
		}
		line1 = desc[:breakPoint]
		line2 = strings.TrimSpace(desc[breakPoint:])

		// Truncate line2 if too long
		if len(line2) > maxWidth {
			line2 = line2[:maxWidth-3] + "..."
		}
	}

	descContent := style.Render(line1)
	if line2 != "" {
		descContent += "\n" + style.Render(line2)
	}

	content := descContent + "\n" + sourceStyle.Render(timeStr+" • "+source)
	return cardStyle.Render(content)
}
