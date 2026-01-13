package reminder

import (
	"sort"
	"time"
)

// Status represents the current state of a reminder
type Status int

const (
	Pending      Status = iota // Waiting for trigger time
	Triggered                  // Time reached, needs acknowledgment
	Acknowledged               // User dismissed, show crossed out
)

func (s Status) String() string {
	switch s {
	case Pending:
		return "pending"
	case Triggered:
		return "TRIGGERED"
	case Acknowledged:
		return "done"
	default:
		return "unknown"
	}
}

// Reminder represents a single reminder parsed from markdown
type Reminder struct {
	DateTime    time.Time
	Description string
	SourceFile  string // For future multi-file support
	LineNumber  int    // Helps user find it in their markdown
	Status      Status
}

// IsDue returns true if the reminder's time has passed
func (r *Reminder) IsDue() bool {
	return time.Now().After(r.DateTime)
}

// SortByDateTime sorts a slice of reminders by their DateTime
func SortByDateTime(reminders []*Reminder) {
	sort.Slice(reminders, func(i, j int) bool {
		return reminders[i].DateTime.Before(reminders[j].DateTime)
	})
}
