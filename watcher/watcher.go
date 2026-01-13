package watcher

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"

	"go_remind/parser"
	"go_remind/reminder"
)

// FileEvent is sent when files are updated with new reminders
type FileEvent struct {
	FilePath  string
	Reminders []*reminder.Reminder
	Err       error
}

// Watcher watches files/directories for changes and parses reminders
type Watcher struct {
	fsWatcher *fsnotify.Watcher
	Events    chan FileEvent
	done      chan struct{}
}

// New creates a new Watcher
func New() (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	return &Watcher{
		fsWatcher: fsw,
		Events:    make(chan FileEvent, 10),
		done:      make(chan struct{}),
	}, nil
}

// WatchFile adds a single file to the watch list
func (w *Watcher) WatchFile(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	return w.fsWatcher.Add(absPath)
}

// WatchDirectory adds all markdown files in a directory to the watch list
func (w *Watcher) WatchDirectory(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	// Walk directory and watch all .md files and subdirectories
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Watch all directories for new files
			if err := w.fsWatcher.Add(path); err != nil {
				log.Printf("Warning: could not watch directory %s: %v", path, err)
			}
		} else if filepath.Ext(path) == ".md" {
			if err := w.fsWatcher.Add(path); err != nil {
				log.Printf("Warning: could not watch %s: %v", path, err)
			}
		}
		return nil
	})
	return err
}

// Start begins watching for file changes
func (w *Watcher) Start() {
	go w.run()
}

// Stop stops the watcher
func (w *Watcher) Stop() {
	close(w.done)
	w.fsWatcher.Close()
}

func (w *Watcher) run() {
	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}

			// Only care about write events
			if !event.Has(fsnotify.Write) && !event.Has(fsnotify.Create) {
				continue
			}

			// Only process markdown files
			if filepath.Ext(event.Name) != ".md" {
				// If it's a new directory or .md file, add it to watch list
				if event.Has(fsnotify.Create) {
					info, err := os.Stat(event.Name)
					if err == nil {
						if info.IsDir() {
							// Watch new directory and all its .md files
							w.WatchDirectory(event.Name)
						} else if filepath.Ext(event.Name) == ".md" {
							w.fsWatcher.Add(event.Name)
						}
					}
				}
				continue
			}

			// Parse the file
			reminders, err := parser.ParseFile(event.Name, time.Now())
			w.Events <- FileEvent{
				FilePath:  event.Name,
				Reminders: reminders,
				Err:       err,
			}

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// ParseInitial parses a file or directory and returns initial reminders
func ParseInitial(path string) ([]*reminder.Reminder, bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false, err
	}

	now := time.Now()
	isDir := info.IsDir()

	if !isDir {
		reminders, err := parser.ParseFile(path, now)
		return reminders, false, err
	}

	// It's a directory - parse all .md files
	var allReminders []*reminder.Reminder
	err = filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(filePath) == ".md" {
			reminders, parseErr := parser.ParseFile(filePath, now)
			if parseErr != nil {
				log.Printf("Warning: could not parse %s: %v", filePath, parseErr)
				return nil // Continue with other files
			}
			allReminders = append(allReminders, reminders...)
		}
		return nil
	})

	return allReminders, true, err
}
