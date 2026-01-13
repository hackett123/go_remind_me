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
	var reminders []*reminder.Reminder
	var tuiEvents chan tui.FileUpdateMsg

	if len(os.Args) >= 2 {
		// File/directory mode
		path := os.Args[1]

		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
			os.Exit(1)
		}

		// Parse initial reminders
		var isDir bool
		reminders, isDir, err = watcher.ParseInitial(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing: %v\n", err)
			os.Exit(1)
		}

		reminder.SortByDateTime(reminders)

		// Set up file watcher
		w, err := watcher.New()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating watcher: %v\n", err)
			os.Exit(1)
		}
		defer w.Stop()

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

		tuiEvents = make(chan tui.FileUpdateMsg, 10)

		w.Start()
		go func() {
			for event := range w.Events {
				if event.Err != nil {
					continue
				}
				tuiEvents <- tui.FileUpdateMsg{
					FilePath:  event.FilePath,
					Reminders: event.Reminders,
				}
			}
			close(tuiEvents)
		}()
	}
	// else: standalone mode - no file watching, empty reminders

	// Run the TUI
	model := tui.New(reminders, tuiEvents)
	p := tea.NewProgram(model)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
