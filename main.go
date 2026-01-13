package main

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"go_remind/reminder"
	"go_remind/tui"
	"go_remind/watcher"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: go_remind <markdown-file-or-directory>")
		os.Exit(1)
	}

	path := os.Args[1]

	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
		os.Exit(1)
	}

	// Parse initial reminders (handles both files and directories)
	reminders, isDir, err := watcher.ParseInitial(absPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing: %v\n", err)
		os.Exit(1)
	}

	// Sort reminders by datetime
	reminder.SortByDateTime(reminders)

	// Set up file watcher
	w, err := watcher.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating watcher: %v\n", err)
		os.Exit(1)
	}
	defer w.Stop()

	// Watch the file or directory
	if isDir {
		if err := w.WatchDirectory(absPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error watching directory: %v\n", err)
			os.Exit(1)
		}
	} else {
		if err := w.WatchFile(absPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error watching file: %v\n", err)
			os.Exit(1)
		}
	}

	// Create channel to send file updates to TUI
	tuiEvents := make(chan tui.FileUpdateMsg, 10)

	// Start the watcher and forward events to TUI channel
	w.Start()
	go func() {
		for event := range w.Events {
			if event.Err != nil {
				continue // Skip errors for now
			}
			tuiEvents <- tui.FileUpdateMsg{
				FilePath:  event.FilePath,
				Reminders: event.Reminders,
			}
		}
		close(tuiEvents)
	}()

	// Run the TUI
	model := tui.New(reminders, tuiEvents)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
