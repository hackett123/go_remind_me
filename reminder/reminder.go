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
	Tags        []string // Tags extracted from content (e.g., #work, #urgent)
	SourceFile  string   // For future multi-file support
	LineNumber  int      // Helps user find it in their markdown
	Status      Status
}

// IsDue returns true if the reminder's time has passed
func (r *Reminder) IsDue() bool {
	return time.Now().After(r.DateTime)
}

// Snoozeable returns true if the reminder can be snoozed
// Acknowledged reminders cannot be snoozed
func (r *Reminder) Snoozeable() bool {
	return r.Status != Acknowledged
}

// SortByDateTime sorts a slice of reminders by their DateTime
func SortByDateTime(reminders []*Reminder) {
	sort.Slice(reminders, func(i, j int) bool {
		return reminders[i].DateTime.Before(reminders[j].DateTime)
	})
}

// MergeFromFile merges new reminders from a file with existing reminders.
// Deduplication is based on (SourceFile, Description):
// - Existing reminders from the same file with matching descriptions are preserved (keeps original DateTime/Status)
// - New reminders with no match are added
// - Pending/triggered reminders from the file that no longer exist are removed
// - Acknowledged reminders are always kept (even if removed from file)
func MergeFromFile(existing []*Reminder, filePath string, newReminders []*Reminder) []*Reminder {
	// Build a map of new reminders by description for quick lookup
	newByDesc := make(map[string]*Reminder)
	for _, r := range newReminders {
		newByDesc[r.Description] = r
	}

	// Build result: start with reminders from OTHER files + acknowledged from this file
	var result []*Reminder
	matchedDescs := make(map[string]bool)

	for _, r := range existing {
		if r.SourceFile != filePath {
			// Keep reminders from other files unchanged
			result = append(result, r)
			continue
		}

		// This reminder is from the file being updated
		if r.Status == Acknowledged {
			// Always keep acknowledged reminders
			result = append(result, r)
			matchedDescs[r.Description] = true
			continue
		}

		// Check if this reminder still exists in the new parse
		if _, exists := newByDesc[r.Description]; exists {
			// Keep the existing reminder (preserves DateTime and Status)
			result = append(result, r)
			matchedDescs[r.Description] = true
		}
		// If not in newByDesc, it was removed from the file - don't include it
	}

	// Add new reminders that weren't matched
	for _, r := range newReminders {
		if !matchedDescs[r.Description] {
			result = append(result, r)
		}
	}

	return result
}
