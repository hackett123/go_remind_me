package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"go_remind/reminder"
)

func (m Model) detailView() string {
	if m.detailReminder == nil {
		return ""
	}

	r := m.detailReminder

	// Detail card
	cardWidth := m.width - 8
	if cardWidth < 40 {
		cardWidth = 40
	}
	if cardWidth > 100 {
		cardWidth = 100
	}

	var statusStyle lipgloss.Style
	switch r.Status {
	case reminder.Triggered:
		statusStyle = triggeredStyle
	case reminder.Acknowledged:
		statusStyle = acknowledgedStyle
	default:
		statusStyle = normalStyle
	}

	detailCardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(statusStyle.GetForeground()).
		Padding(1, 2).
		Width(cardWidth)

	// Content with scrolling
	var content strings.Builder
	content.WriteString(inputLabelStyle.Render("Description:"))
	content.WriteString("\n\n")

	// Wrap description text
	descLines := wrapText(r.Description, cardWidth-4)
	visibleLines := m.height - 15
	if visibleLines < 5 {
		visibleLines = 5
	}

	startLine := m.detailScroll
	endLine := startLine + visibleLines
	if endLine > len(descLines) {
		endLine = len(descLines)
	}
	if startLine >= len(descLines) {
		startLine = len(descLines) - 1
		if startLine < 0 {
			startLine = 0
		}
	}

	for i := startLine; i < endLine; i++ {
		content.WriteString(normalStyle.Render(descLines[i]))
		content.WriteString("\n")
	}

	content.WriteString("\n")
	content.WriteString(sourceStyle.Render("─────────────────────────────────"))
	content.WriteString("\n\n")

	// Metadata
	timeStr := r.DateTime.Format("Monday, January 2, 2006 at 3:04 PM")
	content.WriteString(inputHintStyle.Render("Time: "))
	content.WriteString(normalStyle.Render(timeStr))
	content.WriteString("\n")

	content.WriteString(inputHintStyle.Render("Status: "))
	content.WriteString(statusStyle.Render(r.Status.String()))
	content.WriteString("\n")

	if len(r.Tags) > 0 {
		content.WriteString(inputHintStyle.Render("Tags: "))
		tagStrs := make([]string, len(r.Tags))
		for i, tag := range r.Tags {
			tagStrs[i] = "#" + tag
		}
		content.WriteString(tagStyle.Render(strings.Join(tagStrs, "  ")))
		content.WriteString("\n")
	}

	if r.SourceFile != "" {
		content.WriteString(inputHintStyle.Render("Source: "))
		content.WriteString(sourceStyle.Render(r.SourceFile))
		content.WriteString("\n")
	}

	// Scroll indicator
	if len(descLines) > visibleLines {
		content.WriteString("\n")
		scrollInfo := fmt.Sprintf("(showing lines %d-%d of %d, use ↑/↓ or k/j to scroll)",
			startLine+1, endLine, len(descLines))
		content.WriteString(inputHintStyle.Render(scrollInfo))
	}

	content.WriteString("\n\n")
	content.WriteString(inputHintStyle.Render("Press ESC to close"))

	detailCard := detailCardStyle.Render(content.String())

	// Center the card
	cardStyle := lipgloss.NewStyle().
		Width(m.width).
		Height(m.height).
		AlignHorizontal(lipgloss.Center).
		AlignVertical(lipgloss.Center)

	return cardStyle.Render(detailCard)
}

func wrapText(text string, width int) []string {
	if width < 10 {
		width = 10
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}

	var lines []string
	var currentLine strings.Builder

	for _, word := range words {
		if currentLine.Len() == 0 {
			currentLine.WriteString(word)
		} else if currentLine.Len()+1+len(word) <= width {
			currentLine.WriteString(" ")
			currentLine.WriteString(word)
		} else {
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentLine.WriteString(word)
		}
	}

	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}
