package tui

import (
	"fmt"
	"io"
	"path/filepath"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"go_remind/reminder"
)

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

func (d itemDelegate) Height() int {
	if currentLayout == LayoutCard {
		return 4
	}
	return 1
}

func (d itemDelegate) Spacing() int {
	if currentLayout == LayoutCard {
		return 1
	}
	return 0
}

func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd { return nil }

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(reminderItem)
	if !ok {
		return
	}

	if currentLayout == LayoutCard {
		d.renderCard(w, m, index, i)
	} else {
		d.renderCompact(w, m, index, i)
	}
}

func (d itemDelegate) renderCompact(w io.Writer, m list.Model, index int, i reminderItem) {
	r := i.reminder
	timeStr := r.DateTime.Format("Jan 2 3:04pm")
	source := filepath.Base(r.SourceFile)

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

	// Use more space for description - no truncation, let it wrap naturally
	line := fmt.Sprintf("%s %-18s %-12s %s", statusIcon, timeStr, r.Status.String(), r.Description)
	styledLine := style.Render(line)
	sourcePart := sourceStyle.Render("  " + source)

	fmt.Fprintf(w, "%s%s", styledLine, sourcePart)
}

func (d itemDelegate) renderCard(w io.Writer, m list.Model, index int, i reminderItem) {
	r := i.reminder
	timeStr := r.DateTime.Format("Mon Jan 2 • 3:04pm")
	source := filepath.Base(r.SourceFile)
	isSelected := index == m.Index()

	var style, borderColor lipgloss.Style
	switch r.Status {
	case reminder.Triggered:
		style = triggeredStyle
		borderColor = lipgloss.NewStyle().Foreground(triggeredStyle.GetForeground())
	case reminder.Acknowledged:
		style = acknowledgedStyle
		borderColor = lipgloss.NewStyle().Foreground(acknowledgedStyle.GetForeground())
	default:
		style = normalStyle
		borderColor = lipgloss.NewStyle().Foreground(normalStyle.GetForeground())
	}

	if isSelected {
		borderColor = lipgloss.NewStyle().Foreground(selectedItemStyle.GetForeground())
		if r.Status != reminder.Triggered && r.Status != reminder.Acknowledged {
			style = selectedItemStyle
		}
	}

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor.GetForeground()).
		Padding(0, 1).
		Width(60)

	desc := style.Render(r.Description)
	meta := sourceStyle.Render(timeStr + "  •  " + source + "  •  " + r.Status.String())
	content := desc + "\n" + meta

	fmt.Fprint(w, cardStyle.Render(content))
}

func remindersToItems(reminders []*reminder.Reminder) []list.Item {
	items := make([]list.Item, len(reminders))
	for i, r := range reminders {
		items[i] = reminderItem{reminder: r}
	}
	return items
}
