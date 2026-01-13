package watcher

import (
	"os"
	"testing"
	"time"
)

func TestSpecificFileWatching(t *testing.T) {
	// Test with a generic test file
	filePath := "../test_generic.md"
	dirPath := ".."

	// First test: ParseInitial
	t.Run("ParseInitial", func(t *testing.T) {
		reminders, isDir, err := ParseInitial(filePath)
		if err != nil {
			t.Fatalf("ParseInitial failed: %v", err)
		}
		if isDir {
			t.Error("Expected isDir to be false for single file")
		}

		t.Logf("ParseInitial found %d reminders:", len(reminders))
		for i, r := range reminders {
			t.Logf("%d. Line %d: '%s'", i+1, r.LineNumber, r.Description)
		}

		if len(reminders) != 3 {
			t.Errorf("Expected 3 reminders from ParseInitial, got %d", len(reminders))
		}
	})

	// Second test: File watching
	t.Run("FileWatching", func(t *testing.T) {
		// Create watcher
		w, err := New()
		if err != nil {
			t.Fatalf("Failed to create watcher: %v", err)
		}
		defer w.Stop()

		w.Start()

		// Watch the directory containing the file
		err = w.WatchDirectory(dirPath)
		if err != nil {
			t.Fatalf("Failed to watch directory: %v", err)
		}

		// Small delay to ensure watcher is ready
		time.Sleep(100 * time.Millisecond)

		// Trigger a file change by appending a space and removing it
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		// Write with a space added
		err = os.WriteFile(filePath, append(content, ' '), 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Write back original content
		err = os.WriteFile(filePath, content, 0644)
		if err != nil {
			t.Fatalf("Failed to restore file: %v", err)
		}

		// Wait for file event
		select {
		case event := <-w.Events:
			if event.Err != nil {
				t.Fatalf("Watcher error: %v", event.Err)
			}

			t.Logf("Found %d reminders from watcher:", len(event.Reminders))
			for i, r := range event.Reminders {
				t.Logf("%d. Line %d: '%s'", i+1, r.LineNumber, r.Description)
			}

			// Should find 3 reminders
			expectedCount := 3
			if len(event.Reminders) != expectedCount {
				t.Errorf("Expected %d reminders, got %d", expectedCount, len(event.Reminders))
			}

		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for file event")
		}
	})
}

// Helper function to touch a file (update its modification time)
func touchFile(path string) error {
	now := time.Now()
	return os.Chtimes(path, now, now)
}
