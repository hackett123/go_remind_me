package main

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"go_remind/parser"
	"go_remind/reminder"
	"go_remind/tui"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go_remind <markdown-file>")
		os.Exit(1)
	}

	filepath := os.Args[1]
	now := time.Now()

	// Parse reminders from the file
	reminders, err := parser.ParseFile(filepath, now)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing file: %v\n", err)
		os.Exit(1)
	}

	// Sort reminders by datetime
	reminder.SortByDateTime(reminders)

	// Run the TUI
	model := tui.New(reminders)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
