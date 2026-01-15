package state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"go_remind/reminder"
)

const stateFileName = "reminders_state.json"

// Store handles persistence of reminders to disk
type Store struct {
	path string
}

// NewStore creates a Store with a custom path
func NewStore(path string) *Store {
	return &Store{path: path}
}

// NewDefaultStore creates a Store using the default path (~/.go_remind/reminders_state.json)
func NewDefaultStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(homeDir, ".go_remind")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, err
	}

	return &Store{
		path: filepath.Join(stateDir, stateFileName),
	}, nil
}

// NewTestStore creates a Store using the test path (~/.go_remind/test/reminders_state.json)
func NewTestStore() (*Store, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	stateDir := filepath.Join(homeDir, ".go_remind", "test")
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return nil, err
	}

	return &Store{
		path: filepath.Join(stateDir, stateFileName),
	}, nil
}

// Path returns the store's file path
func (s *Store) Path() string {
	return s.path
}

// savedReminder is the JSON-serializable form of a reminder
type savedReminder struct {
	DateTime    time.Time `json:"datetime"`
	Description string    `json:"description"`
	Tags        []string  `json:"tags,omitempty"`
	SourceFile  string    `json:"source_file"`
	Status      int       `json:"status"`
}

// Load reads reminders from the state file
func (s *Store) Load() ([]*reminder.Reminder, error) {
	data, err := os.ReadFile(s.path)
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
	for i, sr := range saved {
		reminders[i] = &reminder.Reminder{
			DateTime:    sr.DateTime,
			Description: sr.Description,
			Tags:        sr.Tags,
			SourceFile:  sr.SourceFile,
			Status:      reminder.Status(sr.Status),
		}
	}

	return reminders, nil
}

// Save writes reminders to the state file
func (s *Store) Save(reminders []*reminder.Reminder) error {
	saved := make([]savedReminder, len(reminders))
	for i, r := range reminders {
		saved[i] = savedReminder{
			DateTime:    r.DateTime,
			Description: r.Description,
			Tags:        r.Tags,
			SourceFile:  r.SourceFile,
			Status:      int(r.Status),
		}
	}

	data, err := json.MarshalIndent(saved, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.path, data, 0644)
}
