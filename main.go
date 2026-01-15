package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"

	"go_remind/reminder"
	"go_remind/state"
	"go_remind/tui"
	"go_remind/watcher"
)

func main() {
	var reminders []*reminder.Reminder
	var tuiEvents chan tui.FileUpdateMsg

	// Parse flags
	testDir := flag.Bool("test_dir", false, "Use test state directory (~/.go_remind/test/)")
	flag.Parse()

	// Create state store
	var store *state.Store
	var err error
	if *testDir {
		store, err = state.NewTestStore()
	} else {
		store, err = state.NewDefaultStore()
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not create state store: %v\n", err)
	}

	// Load saved state first
	if store != nil {
		savedReminders, err := store.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: could not load state: %v\n", err)
		}
		if savedReminders != nil {
			reminders = savedReminders
		}
	}

	// Get remaining arguments after flags
	args := flag.Args()

	if len(args) >= 1 {
		// File/directory mode
		path := args[0]

		absPath, err := filepath.Abs(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving path: %v\n", err)
			os.Exit(1)
		}

		// Parse reminders from files
		fileReminders, isDir, err := watcher.ParseInitial(absPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing: %v\n", err)
			os.Exit(1)
		}

		// Merge file reminders with saved state
		// File reminders take precedence for deduplication
		for _, fr := range fileReminders {
			reminders = reminder.MergeFromFile(reminders, fr.SourceFile, []*reminder.Reminder{fr})
		}

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

	reminder.SortByDateTime(reminders)

	// Run the TUI
	model := tui.New(reminders, tuiEvents, store)
	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
