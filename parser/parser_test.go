package parser

import (
	"os"
	"testing"
	"time"

	"go_remind/reminder"
)

func TestParseFile(t *testing.T) {
	// Create a temporary file for testing
	tempFile, err := os.CreateTemp("", "parser_test_*.md")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	baseTime := time.Date(2026, 1, 13, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name     string
		content  string
		expected int
		checkFirst func(*testing.T, *reminder.Reminder)
	}{
		{
			name: "single reminder",
			content: `# Test File
This has a [remind_me +1h Test reminder] in it.`,
			expected: 1,
			checkFirst: func(t *testing.T, r *reminder.Reminder) {
				if r.Description != "Test reminder" {
					t.Errorf("Expected description 'Test reminder', got '%s'", r.Description)
				}
				expectedTime := baseTime.Add(time.Hour)
				if !r.DateTime.Equal(expectedTime) {
					t.Errorf("Expected time %v, got %v", expectedTime, r.DateTime)
				}
			},
		},
		{
			name: "multiple reminders same line",
			content: `Multiple [remind_me +1h First] and [remind_me +2h Second] on same line.`,
			expected: 2,
			checkFirst: func(t *testing.T, r *reminder.Reminder) {
				if r.Description != "First" {
					t.Errorf("Expected first description 'First', got '%s'", r.Description)
				}
			},
		},
		{
			name: "multiple reminders different lines",
			content: `# Test
Line 1: [remind_me +1h First reminder]
Line 2: [remind_me +2h Second reminder]
Line 3: [remind_me +3h Third reminder]`,
			expected: 3,
		},
		{
			name: "various datetime formats",
			content: `# Test
Relative: [remind_me +30m Relative time]
Natural: [remind_me tomorrow 9am Natural language]
Specific: [remind_me 2026-01-15T14:30 Specific datetime]
Time only: [remind_me 3pm Time only today]`,
			expected: 4,
		},
		{
			name: "edge cases with whitespace",
			content: `# Test
Extra spaces: [remind_me    +1h   Lots of spaces   ]
Tabs and spaces: [remind_me	+2h	Mixed whitespace	]`,
			expected: 2,
			checkFirst: func(t *testing.T, r *reminder.Reminder) {
				if r.Description != "Lots of spaces" {
					t.Errorf("Expected description 'Lots of spaces', got '%s'", r.Description)
				}
			},
		},
		{
			name: "invalid reminders should be skipped",
			content: `# Test
Valid: [remind_me +1h Valid reminder]
Invalid no desc: [remind_me +1h]
Invalid no time: [remind_me No time here]
Invalid empty: [remind_me]
Another valid: [remind_me +2h Another valid]`,
			expected: 2,
		},
		{
			name: "complex datetime parsing",
			content: `# Test
Multi-word time: [remind_me Jan 15 3:30pm Call dentist]
Weekday: [remind_me friday 10am Team meeting]
Relative with units: [remind_me +1h30m Long meeting]`,
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Write test content
			err := os.WriteFile(tempFile.Name(), []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to write test content: %v", err)
			}

			// Parse the file
			reminders, err := ParseFile(tempFile.Name(), baseTime)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			// Check count
			if len(reminders) != tt.expected {
				t.Errorf("Expected %d reminders, got %d", tt.expected, len(reminders))
				for i, r := range reminders {
					t.Logf("Reminder %d: '%s' at %v (line %d)", i, r.Description, r.DateTime, r.LineNumber)
				}
				return
			}

			// Check that all reminders have required fields
			for i, r := range reminders {
				if r.Description == "" {
					t.Errorf("Reminder %d has empty description", i)
				}
				if r.DateTime.IsZero() {
					t.Errorf("Reminder %d has zero datetime", i)
				}
				if r.SourceFile != tempFile.Name() {
					t.Errorf("Reminder %d has wrong source file: expected %s, got %s", i, tempFile.Name(), r.SourceFile)
				}
				if r.LineNumber <= 0 {
					t.Errorf("Reminder %d has invalid line number: %d", i, r.LineNumber)
				}
				if r.Status != reminder.Pending {
					t.Errorf("Reminder %d has wrong status: expected %v, got %v", i, reminder.Pending, r.Status)
				}
			}

			// Run custom check if provided
			if tt.checkFirst != nil && len(reminders) > 0 {
				tt.checkFirst(t, reminders[0])
			}
		})
	}
}

func TestParseReminderContent(t *testing.T) {
	baseTime := time.Date(2026, 1, 13, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name        string
		content     string
		expectError bool
		expectedDesc string
	}{
		{
			name:         "simple relative time",
			content:      "+1h Test reminder",
			expectError:  false,
			expectedDesc: "Test reminder",
		},
		{
			name:         "multi-word description",
			content:      "+30m This is a longer description",
			expectError:  false,
			expectedDesc: "This is a longer description",
		},
		{
			name:         "complex datetime",
			content:      "Jan 15 3:30pm Call the dentist",
			expectError:  false,
			expectedDesc: "Call the dentist",
		},
		{
			name:        "no description",
			content:     "+1h",
			expectError: true,
		},
		{
			name:        "no datetime",
			content:     "Just description",
			expectError: true,
		},
		{
			name:        "empty content",
			content:     "",
			expectError: true,
		},
		{
			name:        "single word",
			content:     "single",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := parseReminderContent(tt.content, baseTime)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if r.Description != tt.expectedDesc {
				t.Errorf("Expected description '%s', got '%s'", tt.expectedDesc, r.Description)
			}

			if r.Status != reminder.Pending {
				t.Errorf("Expected status Pending, got %v", r.Status)
			}
		})
	}
}

func TestExtractTags(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		expectedText string
		expectedTags []string
	}{
		{
			name:         "single tag",
			input:        "Call mom #family",
			expectedText: "Call mom",
			expectedTags: []string{"family"},
		},
		{
			name:         "multiple tags",
			input:        "Meeting #work #important",
			expectedText: "Meeting",
			expectedTags: []string{"work", "important"},
		},
		{
			name:         "no tags",
			input:        "Call mom",
			expectedText: "Call mom",
			expectedTags: nil,
		},
		{
			name:         "tag in middle",
			input:        "Call #family mom today",
			expectedText: "Call mom today",
			expectedTags: []string{"family"},
		},
		{
			name:         "tag at start",
			input:        "#urgent Call mom",
			expectedText: "Call mom",
			expectedTags: []string{"urgent"},
		},
		{
			name:         "adjacent hash not a tag",
			input:        "Check item#urgent later",
			expectedText: "Check item#urgent later",
			expectedTags: nil,
		},
		{
			name:         "multiple tags scattered",
			input:        "#urgent Call #family mom #important",
			expectedText: "Call mom",
			expectedTags: []string{"urgent", "family", "important"},
		},
		{
			name:         "tag with numbers",
			input:        "Review PR #pr123",
			expectedText: "Review PR",
			expectedTags: []string{"pr123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanText, tags := ExtractTags(tt.input)

			if cleanText != tt.expectedText {
				t.Errorf("Expected text '%s', got '%s'", tt.expectedText, cleanText)
			}

			if len(tags) != len(tt.expectedTags) {
				t.Errorf("Expected %d tags, got %d: %v", len(tt.expectedTags), len(tags), tags)
				return
			}

			for i, tag := range tags {
				if tag != tt.expectedTags[i] {
					t.Errorf("Expected tag[%d] '%s', got '%s'", i, tt.expectedTags[i], tag)
				}
			}
		})
	}
}

func TestParseReminderContentWithTags(t *testing.T) {
	baseTime := time.Date(2026, 1, 13, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name         string
		content      string
		expectedDesc string
		expectedTags []string
	}{
		{
			name:         "reminder with single tag",
			content:      "+1h Call mom #family",
			expectedDesc: "Call mom",
			expectedTags: []string{"family"},
		},
		{
			name:         "reminder with multiple tags",
			content:      "+30m Meeting #work #important",
			expectedDesc: "Meeting",
			expectedTags: []string{"work", "important"},
		},
		{
			name:         "reminder without tags",
			content:      "+1h Call mom",
			expectedDesc: "Call mom",
			expectedTags: nil,
		},
		{
			name:         "complex datetime with tags",
			content:      "Jan 15 3:30pm Dentist appointment #health #important",
			expectedDesc: "Dentist appointment",
			expectedTags: []string{"health", "important"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := parseReminderContent(tt.content, baseTime)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if r.Description != tt.expectedDesc {
				t.Errorf("Expected description '%s', got '%s'", tt.expectedDesc, r.Description)
			}

			if len(r.Tags) != len(tt.expectedTags) {
				t.Errorf("Expected %d tags, got %d: %v", len(tt.expectedTags), len(r.Tags), r.Tags)
				return
			}

			for i, tag := range r.Tags {
				if tag != tt.expectedTags[i] {
					t.Errorf("Expected tag[%d] '%s', got '%s'", i, tt.expectedTags[i], tag)
				}
			}
		})
	}
}

func TestRegexPattern(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "single match",
			input:    "This has [remind_me +1h test] in it",
			expected: 1,
		},
		{
			name:     "multiple matches",
			input:    "First [remind_me +1h one] and [remind_me +2h two] here",
			expected: 2,
		},
		{
			name:     "no matches",
			input:    "This has no reminders in it",
			expected: 0,
		},
		{
			name:     "malformed brackets",
			input:    "This has [remind_me +1h unclosed and [remind_me +2h test] closed",
			expected: 1,
		},
		{
			name:     "nested brackets",
			input:    "This has [remind_me +1h [nested] content] here",
			expected: 1, // Matches up to first closing bracket: "[nested"
		},
		{
			name:     "extra whitespace",
			input:    "This has [remind_me    +1h   lots of spaces   ] here",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := remindPattern.FindAllStringSubmatch(tt.input, -1)
			if len(matches) != tt.expected {
				t.Errorf("Expected %d matches, got %d", tt.expected, len(matches))
				for i, match := range matches {
					t.Logf("Match %d: %v", i, match)
				}
			}
		})
	}
}
