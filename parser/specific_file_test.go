package parser

import (
	"testing"
	"time"
)

func TestSpecificFileIssue(t *testing.T) {
	// Test with a generic test file
	filePath := "../test_generic.md"
	baseTime := time.Date(2026, 1, 13, 13, 27, 0, 0, time.Local)

	reminders, err := ParseFile(filePath, baseTime)
	if err != nil {
		t.Fatalf("Failed to parse file %s: %v", filePath, err)
	}

	t.Logf("Found %d reminders in %s:", len(reminders), filePath)
	for i, r := range reminders {
		t.Logf("%d. Line %d: '%s' at %v", i+1, r.LineNumber, r.Description, r.DateTime)
	}

	// Expected reminders:
	// 1. "Complete project setup" (+1h)
	// 2. "Review code changes" (+10m) 
	// 3. "Send update email" (+5m)
	expectedCount := 3
	if len(reminders) != expectedCount {
		t.Errorf("Expected %d reminders, got %d", expectedCount, len(reminders))
	}

	// Verify specific reminders exist
	expectedDescriptions := map[string]bool{
		"Complete project setup": false,
		"Review code changes":    false,
		"Send update email":      false,
	}

	for _, r := range reminders {
		if _, exists := expectedDescriptions[r.Description]; exists {
			expectedDescriptions[r.Description] = true
		}
	}

	for desc, found := range expectedDescriptions {
		if !found {
			t.Errorf("Expected to find reminder: '%s'", desc)
		}
	}
}
