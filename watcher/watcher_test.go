package watcher

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"go_remind/reminder"
)

func TestWatcherFileUpdates(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "single reminder",
			content: `# Test
This has [remind_me +1h Test reminder] in it.`,
			expected: 1,
		},
		{
			name: "multiple reminders same line",
			content: `# Test
Multiple [remind_me +1h First] and [remind_me +2h Second] reminders.`,
			expected: 2,
		},
		{
			name: "multiple reminders different lines",
			content: `# Test
First [remind_me +1h First reminder] here.
Second [remind_me +2h Second reminder] there.
Third [remind_me +3h Third reminder] everywhere.`,
			expected: 3,
		},
		{
			name: "complex datetime formats",
			content: `# Test
Relative: [remind_me +30m Relative time]
Natural: [remind_me tomorrow 9am Natural language]
Specific: [remind_me 2026-01-15T14:30 Specific datetime]
Time only: [remind_me 3pm Time only today]`,
			expected: 4,
		},
		{
			name: "edge cases",
			content: `# Test
Spaces: [remind_me    +1h   Lots of spaces   ]
Mixed: [remind_me Jan 15 3pm Call mom] normal text [remind_me +2h Another]
Empty line above

[remind_me +5m After empty line]`,
			expected: 4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a fresh temp directory and watcher for each test
			tempDir, err := os.MkdirTemp("", "watcher_test")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			w, err := New()
			if err != nil {
				t.Fatalf("Failed to create watcher: %v", err)
			}
			defer w.Stop()

			w.Start()

			err = w.WatchDirectory(tempDir)
			if err != nil {
				t.Fatalf("Failed to watch directory: %v", err)
			}

			// Create the test file
			testFile := filepath.Join(tempDir, "test.md")
			err = os.WriteFile(testFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test file: %v", err)
			}

			// Wait for file event
			select {
			case event := <-w.Events:
				if event.Err != nil {
					t.Fatalf("Watcher error: %v", event.Err)
				}
				if len(event.Reminders) != tt.expected {
					t.Errorf("Expected %d reminders, got %d", tt.expected, len(event.Reminders))
					for i, r := range event.Reminders {
						t.Logf("Reminder %d: %s at %v", i, r.Description, r.DateTime)
					}
				}
				if event.FilePath != testFile {
					t.Errorf("Expected file path %s, got %s", testFile, event.FilePath)
				}
			case <-time.After(2 * time.Second):
				t.Fatal("Timeout waiting for file event")
			}
		})
	}
}

func TestWatcherNewFileCreation(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "watcher_new_file_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create watcher
	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	w.Start()

	// Watch the directory
	err = w.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

	// Create a new markdown file
	newFile := filepath.Join(tempDir, "new_file.md")
	content := `# New File
This is a [remind_me +1h New file reminder] in a newly created file.`

	err = os.WriteFile(newFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create new file: %v", err)
	}

	// Wait for file event
	select {
	case event := <-w.Events:
		if event.Err != nil {
			t.Fatalf("Watcher error: %v", event.Err)
		}
		if len(event.Reminders) != 1 {
			t.Errorf("Expected 1 reminder, got %d", len(event.Reminders))
		}
		if event.FilePath != newFile {
			t.Errorf("Expected file path %s, got %s", newFile, event.FilePath)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for new file event")
	}
}

func TestParseInitialDirectory(t *testing.T) {
	// Create temporary directory with multiple files
	tempDir, err := os.MkdirTemp("", "parse_initial_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create test files
	files := map[string]string{
		"file1.md": `# File 1
First [remind_me +1h File 1 reminder] here.`,
		"file2.md": `# File 2
Second [remind_me +2h File 2 reminder] there.
Another [remind_me +3h Another file 2 reminder] one.`,
		"file3.txt": `This is not markdown and should be ignored.`,
		"subdir/file4.md": `# File 4
Nested [remind_me +4h Nested reminder] file.`,
	}

	for filename, content := range files {
		fullPath := filepath.Join(tempDir, filename)
		err := os.MkdirAll(filepath.Dir(fullPath), 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", filename, err)
		}
	}

	// Parse initial directory
	reminders, isDir, err := ParseInitial(tempDir)
	if err != nil {
		t.Fatalf("Failed to parse initial directory: %v", err)
	}

	if !isDir {
		t.Error("Expected isDir to be true")
	}

	// Should find 4 reminders (1 + 2 + 1, ignoring .txt file)
	if len(reminders) != 4 {
		t.Errorf("Expected 4 reminders, got %d", len(reminders))
		for i, r := range reminders {
			t.Logf("Reminder %d: %s from %s", i, r.Description, r.SourceFile)
		}
	}

	// Verify source files are set correctly
	sourceFiles := make(map[string]int)
	for _, r := range reminders {
		if r.SourceFile == "" {
			t.Error("Reminder missing source file")
		}
		sourceFiles[filepath.Base(r.SourceFile)]++
	}

	expectedFiles := map[string]int{
		"file1.md": 1,
		"file2.md": 2,
		"file4.md": 1,
	}

	for file, expectedCount := range expectedFiles {
		if count, exists := sourceFiles[file]; !exists || count != expectedCount {
			t.Errorf("Expected %d reminders from %s, got %d", expectedCount, file, count)
		}
	}
}

func TestParseInitialSingleFile(t *testing.T) {
	// Create temporary file
	tempFile, err := os.CreateTemp("", "parse_single_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	content := `# Single File Test
This has [remind_me +1h Single file reminder] in it.
And [remind_me +2h Another single file reminder] too.`

	_, err = tempFile.WriteString(content)
	if err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tempFile.Close()

	// Parse single file
	reminders, isDir, err := ParseInitial(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to parse single file: %v", err)
	}

	if isDir {
		t.Error("Expected isDir to be false for single file")
	}

	if len(reminders) != 2 {
		t.Errorf("Expected 2 reminders, got %d", len(reminders))
	}

	for _, r := range reminders {
		if r.SourceFile != tempFile.Name() {
			t.Errorf("Expected source file %s, got %s", tempFile.Name(), r.SourceFile)
		}
		if r.Status != reminder.Pending {
			t.Errorf("Expected status Pending, got %v", r.Status)
		}
	}
}

func TestWatcherIgnoresNonMarkdownFiles(t *testing.T) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "watcher_ignore_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create watcher
	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	w.Start()

	// Watch the directory
	err = w.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

	// Create a non-markdown file first
	txtFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(txtFile, []byte("This has [remind_me +1h Should be ignored] but it's not markdown."), 0644)
	if err != nil {
		t.Fatalf("Failed to write txt file: %v", err)
	}

	// Small delay to ensure txt file event is processed (and ignored) first
	time.Sleep(100 * time.Millisecond)

	// Create a markdown file
	mdFile := filepath.Join(tempDir, "test.md")
	err = os.WriteFile(mdFile, []byte("This has [remind_me +1h Should be detected] and it's markdown."), 0644)
	if err != nil {
		t.Fatalf("Failed to write md file: %v", err)
	}

	// Collect events - file creation may generate multiple events (CREATE + WRITE)
	// We just want to verify all events are for the .md file, not .txt
	gotMdEvent := false
	timeout := time.After(2 * time.Second)

	for {
		select {
		case event := <-w.Events:
			if event.Err != nil {
				t.Fatalf("Watcher error: %v", event.Err)
			}
			// Verify all events are for .md file, not .txt
			if event.FilePath == txtFile {
				t.Errorf("Got event for .txt file, should be ignored: %+v", event)
			}
			if event.FilePath == mdFile {
				gotMdEvent = true
				if len(event.Reminders) != 1 {
					t.Errorf("Expected 1 reminder, got %d", len(event.Reminders))
				}
			}
		case <-time.After(300 * time.Millisecond):
			// No more events after short wait
			if !gotMdEvent {
				t.Fatal("Never got event for markdown file")
			}
			return
		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}
}

func TestWatchSingleFileMultipleUpdates(t *testing.T) {
	// Create a temp file
	tempFile, err := os.CreateTemp("", "watch_multi_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)
	tempFile.Close()

	// Create watcher and watch the single file
	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	err = w.WatchFile(tempPath)
	if err != nil {
		t.Fatalf("Failed to watch file: %v", err)
	}

	w.Start()

	// Give watcher time to set up
	time.Sleep(100 * time.Millisecond)

	// Perform multiple updates
	updates := []struct {
		content  string
		expected int
	}{
		{"[remind_me +1h One]", 1},
		{"[remind_me +1h One]\n[remind_me +2h Two]", 2},
		{"[remind_me +1h One]\n[remind_me +2h Two]\n[remind_me +3h Three]", 3},
	}

	for i, update := range updates {
		err = os.WriteFile(tempPath, []byte(update.content), 0644)
		if err != nil {
			t.Fatalf("Update %d: Failed to write file: %v", i, err)
		}

		// Wait for event
		select {
		case event := <-w.Events:
			if event.Err != nil {
				t.Fatalf("Update %d: Watcher error: %v", i, event.Err)
			}
			if len(event.Reminders) != update.expected {
				t.Errorf("Update %d: Expected %d reminders, got %d", i, update.expected, len(event.Reminders))
			}
			t.Logf("Update %d: Got %d reminders as expected", i, len(event.Reminders))
		case <-time.After(3 * time.Second):
			t.Fatalf("Update %d: Timeout waiting for event", i)
		}

		// Small delay between updates
		time.Sleep(100 * time.Millisecond)
	}
}

func TestWatchSingleFileEditorSimulation(t *testing.T) {
	// Simulates how editors typically save files:
	// 1. Write to temp file
	// 2. Rename temp to target (or write directly with truncate)

	tempFile, err := os.CreateTemp("", "watch_editor_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	// Write initial content
	initialContent := "[remind_me +1h Initial reminder]"
	_, err = tempFile.WriteString(initialContent)
	if err != nil {
		t.Fatalf("Failed to write initial content: %v", err)
	}
	tempFile.Close()

	// Create watcher
	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	err = w.WatchFile(tempPath)
	if err != nil {
		t.Fatalf("Failed to watch file: %v", err)
	}

	w.Start()
	time.Sleep(100 * time.Millisecond)

	// Simulate editor save: open, truncate, write, close
	f, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		t.Fatalf("Failed to open file for writing: %v", err)
	}
	newContent := `[remind_me +1h Updated first]
[remind_me +2h New second]`
	_, err = f.WriteString(newContent)
	if err != nil {
		t.Fatalf("Failed to write: %v", err)
	}
	f.Close()

	// Wait for event
	select {
	case event := <-w.Events:
		if event.Err != nil {
			t.Fatalf("Watcher error: %v", event.Err)
		}
		if len(event.Reminders) != 2 {
			t.Errorf("Expected 2 reminders, got %d", len(event.Reminders))
			for i, r := range event.Reminders {
				t.Logf("Reminder %d: %s", i, r.Description)
			}
		}
	case <-time.After(3 * time.Second):
		t.Fatal("Timeout waiting for event")
	}
}

func TestWatcherRecursiveDirectories(t *testing.T) {
	// Create temporary directory structure
	tempDir, err := os.MkdirTemp("", "watcher_recursive_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create subdirectories
	subDir1 := filepath.Join(tempDir, "subdir1")
	subDir2 := filepath.Join(tempDir, "subdir1", "nested")
	err = os.MkdirAll(subDir2, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdirectories: %v", err)
	}

	// Create watcher
	w, err := New()
	if err != nil {
		t.Fatalf("Failed to create watcher: %v", err)
	}
	defer w.Stop()

	w.Start()

	// Watch the root directory (should be recursive)
	err = w.WatchDirectory(tempDir)
	if err != nil {
		t.Fatalf("Failed to watch directory: %v", err)
	}

	// Create files in different levels
	files := map[string]string{
		filepath.Join(tempDir, "root.md"):     "[remind_me +1h Root level]",
		filepath.Join(subDir1, "sub1.md"):     "[remind_me +2h Sub level 1]",
		filepath.Join(subDir2, "nested.md"):   "[remind_me +3h Nested level]",
	}

	expectedEvents := len(files)
	receivedEvents := 0

	for filePath, content := range files {
		err = os.WriteFile(filePath, []byte(content), 0644)
		if err != nil {
			t.Fatalf("Failed to write file %s: %v", filePath, err)
		}
	}

	// Collect all events
	timeout := time.After(3 * time.Second)
	for receivedEvents < expectedEvents {
		select {
		case event := <-w.Events:
			if event.Err != nil {
				t.Fatalf("Watcher error: %v", event.Err)
			}
			if len(event.Reminders) != 1 {
				t.Errorf("Expected 1 reminder from %s, got %d", event.FilePath, len(event.Reminders))
			}
			receivedEvents++
			t.Logf("Received event for %s: %s", event.FilePath, event.Reminders[0].Description)
		case <-timeout:
			t.Fatalf("Timeout waiting for events. Received %d/%d events", receivedEvents, expectedEvents)
		}
	}

	if receivedEvents != expectedEvents {
		t.Errorf("Expected %d events, got %d", expectedEvents, receivedEvents)
	}
}
