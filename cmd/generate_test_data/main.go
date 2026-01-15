package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type savedReminder struct {
	DateTime    time.Time `json:"datetime"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags,omitempty"`
	SourceFile  string    `json:"source_file"`
	Status      int       `json:"status"`
}

var tags = []string{"work", "personal", "urgent", "meeting", "followup"}

var descriptions = []string{
	"Team standup meeting",
	"Review pull request",
	"Submit expense report",
	"Update project documentation",
	"Call with client",
	"Code review session",
	"Sprint planning",
	"Deploy to production",
	"Database backup check",
	"Security audit review",
	"Performance testing",
	"Bug triage meeting",
	"1:1 with manager",
	"Write unit tests",
	"Update dependencies",
	"Refactor legacy code",
	"API design review",
	"Infrastructure planning",
	"Release notes draft",
	"Customer feedback review",
	"Technical debt discussion",
	"Architecture review",
	"Onboarding new team member",
	"Knowledge sharing session",
	"Quarterly planning",
	"Budget review",
	"Vendor evaluation",
	"System health check",
	"Backup verification",
	"Certificate renewal",
	"License renewal check",
	"Capacity planning",
	"Incident postmortem",
	"Documentation update",
	"Training session",
	"Team retrospective",
	"Feature demo",
	"Stakeholder update",
	"Risk assessment",
	"Compliance review",
}

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting home dir: %v\n", err)
		os.Exit(1)
	}

	testDir := filepath.Join(homeDir, ".go_remind", "test")
	if err := os.MkdirAll(testDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating test dir: %v\n", err)
		os.Exit(1)
	}

	now := time.Now()
	// Range from 30 days ago to 1 year from now
	pastDays := 30
	futureDays := 365
	totalDays := pastDays + futureDays

	reminders := make([]savedReminder, 200)
	for i := 0; i < 200; i++ {
		// Random day offset from -30 days to +365 days
		dayOffset := rand.Intn(totalDays+1) - pastDays
		// Random hour (8am to 6pm for realistic business hours)
		hour := 8 + rand.Intn(11)
		// Random minute (on the hour, :15, :30, or :45)
		minute := rand.Intn(4) * 15

		reminderTime := time.Date(
			now.Year(), now.Month(), now.Day()+dayOffset,
			hour, minute, 0, 0, now.Location(),
		)

		desc := descriptions[rand.Intn(len(descriptions))]
		// Add a number to make descriptions unique
		desc = fmt.Sprintf("%s (%d)", desc, i+1)

		// Generate 0-5 random tags
		numTags := rand.Intn(6)
		var reminderTags []string
		if numTags > 0 {
			// Shuffle and pick first numTags
			perm := rand.Perm(len(tags))
			for j := 0; j < numTags; j++ {
				reminderTags = append(reminderTags, tags[perm[j]])
			}
		}

		status := 0 // Pending
		// Make some past reminders triggered or acknowledged
		if reminderTime.Before(now) {
			if rand.Float32() < 0.5 {
				status = 1 // Triggered
			} else {
				status = 2 // Acknowledged
			}
		}

		reminders[i] = savedReminder{
			DateTime:    reminderTime,
			Description: desc,
			Tags:        reminderTags,
			SourceFile:  "test_generated",
			Status:      status,
		}
	}

	data, err := json.MarshalIndent(reminders, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling JSON: %v\n", err)
		os.Exit(1)
	}

	statePath := filepath.Join(testDir, "reminders_state.json")
	if err := os.WriteFile(statePath, data, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated 200 test reminders at %s\n", statePath)
}
