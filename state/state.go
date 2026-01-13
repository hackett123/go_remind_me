package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"go_remind/reminder"
)

const stateFileName = "reminders_state.json"

// savedReminder is the JSON-serializable form of a reminder
type savedReminder struct {
	DateTime    time.Time `json:"datetime"`
	Description string    `json:"description"`
	SourceFile  string    `json:"source_file"`
	Status      int       `json:"status"`
}

// GetStatePath returns the path to the state file
func GetStatePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	stateDir := filepath.Join(homeDir, ".go_remind")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return "", err
	}

	return filepath.Join(stateDir, stateFileName), nil
}

// Load reads reminders from the state file
func Load() ([]*reminder.Reminder, error) {
	path, err := GetStatePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No state file yet, that's OK
		}
		return nil, err
	}

	var saved []savedReminder
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, err
	}

	reminders := make([]*reminder.Reminder, len(saved))
	for i, s := range saved {
		reminders[i] = &reminder.Reminder{
			DateTime:    s.DateTime,
			Description: s.Description,
			SourceFile:  s.SourceFile,
			Status:      reminder.Status(s.Status),
		}
	}

	return reminders, nil
}

// Save writes reminders to the state file
func Save(reminders []*reminder.Reminder) error {
	path, err := GetStatePath()
	if err != nil {
		return err
	}

	saved := make([]savedReminder, len(reminders))
	for i, r := range reminders {
		saved[i] = savedReminder{
			DateTime:    r.DateTime,
			Description: r.Description,
			SourceFile:  r.SourceFile,
			Status:      int(r.Status),
		}
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}
