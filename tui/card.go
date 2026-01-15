package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"go_remind/reminder"
)

func (m Model) gridViewContent() string {
	items := m.getFilteredReminders()
	if len(items) == 0 {
		return normalStyle.Render("No reminders")
	}

	cardWidth := 38
	cols := m.gridColumns
	if cols < 1 {
		cols = 1
	}

	visibleRows := m.visibleGridRows()
	totalRows := (len(items) + cols - 1) / cols // ceiling division

	if !m.sortEnabled {
		// No sorting - render only visible rows
		var rows []string

		// Add scroll up indicator
		if m.gridScroll > 0 {
			rows = append(rows, sourceStyle.Render(fmt.Sprintf("  ↑ %d more rows above", m.gridScroll)))
		}

		startRow := m.gridScroll
		endRow := m.gridScroll + visibleRows
		if endRow > totalRows {
			endRow = totalRows
		}

		for row := startRow; row < endRow; row++ {
			var rowCards []string
			for col := 0; col < cols; col++ {
				idx := row*cols + col
				if idx >= len(items) {
					break
				}
				rowCards = append(rowCards, m.renderCard(items[idx], idx, cardWidth))
			}
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
		}

		// Add scroll down indicator
		if endRow < totalRows {
			rows = append(rows, sourceStyle.Render(fmt.Sprintf("  ↓ %d more rows below", totalRows-endRow)))
		}

		return lipgloss.JoinVertical(lipgloss.Left, rows...)
	}

	// Sort into sections with proper row tracking
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
		MarginBottom(1)

	var sections []string
	globalIdx := 0
	currentRow := 0

	// Add scroll up indicator for sorted view
	if m.gridScroll > 0 {
		sections = append(sections, sourceStyle.Render(fmt.Sprintf("  ↑ %d more rows above", m.gridScroll)))
	}

	// Helper to add a section
	addSection := func(items []*reminder.Reminder, title string) {
		if len(items) > 0 {
			header, content, newRow, newIdx := m.renderSectionWithRowTracking(items, title, sectionStyle, globalIdx, currentRow, cols, cardWidth)
			if header != "" {
				sections = append(sections, header)
			}
			if content != "" {
				sections = append(sections, content)
			}
			currentRow = newRow
			globalIdx = newIdx
		}
	}

	addSection(due, "Due")
	addSection(comingUp, "Coming Up!")
	addSection(tomorrow, "Tomorrow")
	addSection(laterThisWeek, "Later This Week")
	addSection(nextWeek, "Next Week")
	addSection(laterThisMonth, "Later This Month")
	addSection(beyondNextMonth, "Next Month & Beyond")

	// Add scroll down indicator
	if m.gridScroll+visibleRows < totalRows {
		sections = append(sections, sourceStyle.Render(fmt.Sprintf("  ↓ %d more rows below", totalRows-m.gridScroll-visibleRows)))
	}

	if len(sections) == 0 {
		return normalStyle.Render("No pending reminders")
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderSectionWithRowTracking renders a section and tracks rows explicitly
// Returns: header (if visible), content, new row number, new global index
func (m Model) renderSectionWithRowTracking(items []*reminder.Reminder, title string, sectionStyle lipgloss.Style, globalIdx, startRow, cols, cardWidth int) (string, string, int, int) {
	var rows []string
	visibleRows := m.visibleGridRows()
	currentRow := startRow
	hasVisibleRows := false

	for i := 0; i < len(items); i += cols {
		var rowCards []string
		for j := 0; j < cols && i+j < len(items); j++ {
			rowCards = append(rowCards, m.renderCard(items[i+j], globalIdx, cardWidth))
			globalIdx++
		}

		// Only include row if it's in the visible range
		if currentRow >= m.gridScroll && currentRow < m.gridScroll+visibleRows {
			rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, rowCards...))
			hasVisibleRows = true
		}
		currentRow++
	}

	header := ""
	if hasVisibleRows {
		header = sectionStyle.Render(title)
	}

	return header, lipgloss.JoinVertical(lipgloss.Left, rows...), currentRow, globalIdx
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
