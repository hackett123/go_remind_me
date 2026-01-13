package datetime

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	// Fixed reference time: Tuesday, January 13, 2026 at 10:00am
	ref := time.Date(2026, 1, 13, 10, 0, 0, 0, time.Local)

	tests := []struct {
		name     string
		input    string
		wantTime time.Time
		wantErr  bool
	}{
		// Relative times
		{
			name:     "relative minutes",
			input:    "+30m",
			wantTime: ref.Add(30 * time.Minute),
		},
		{
			name:     "relative hours",
			input:    "+2h",
			wantTime: ref.Add(2 * time.Hour),
		},
		{
			name:     "relative days",
			input:    "+1d",
			wantTime: ref.Add(24 * time.Hour),
		},
		{
			name:     "relative combined",
			input:    "+1h30m",
			wantTime: ref.Add(1*time.Hour + 30*time.Minute),
		},

		// Tomorrow
		{
			name:     "tomorrow default 9am",
			input:    "tomorrow",
			wantTime: time.Date(2026, 1, 14, 9, 0, 0, 0, time.Local),
		},
		{
			name:     "tomorrow with time",
			input:    "tomorrow 3pm",
			wantTime: time.Date(2026, 1, 14, 15, 0, 0, 0, time.Local),
		},
		{
			name:     "tomorrow with time minutes",
			input:    "tomorrow 3:30pm",
			wantTime: time.Date(2026, 1, 14, 15, 30, 0, 0, time.Local),
		},

		// In X days/hours
		{
			name:     "in 3 days",
			input:    "in 3 days",
			wantTime: ref.AddDate(0, 0, 3),
		},
		{
			name:     "in 1 day",
			input:    "in 1 day",
			wantTime: ref.AddDate(0, 0, 1),
		},
		{
			name:     "in 2 hours",
			input:    "in 2 hours",
			wantTime: ref.Add(2 * time.Hour),
		},
		{
			name:     "in 30 minutes",
			input:    "in 30 minutes",
			wantTime: ref.Add(30 * time.Minute),
		},

		// Weekday (ref is Tuesday Jan 13)
		{
			name:     "friday default 9am",
			input:    "friday",
			wantTime: time.Date(2026, 1, 16, 9, 0, 0, 0, time.Local), // Friday Jan 16
		},
		{
			name:     "friday with time",
			input:    "friday 10am",
			wantTime: time.Date(2026, 1, 16, 10, 0, 0, 0, time.Local),
		},
		{
			name:     "fri abbreviated",
			input:    "fri 3pm",
			wantTime: time.Date(2026, 1, 16, 15, 0, 0, 0, time.Local),
		},
		{
			name:     "monday next week",
			input:    "monday",
			wantTime: time.Date(2026, 1, 19, 9, 0, 0, 0, time.Local), // Monday Jan 19
		},
		{
			name:     "tuesday next week (same day)",
			input:    "tuesday",
			wantTime: time.Date(2026, 1, 20, 9, 0, 0, 0, time.Local), // Next Tuesday Jan 20
		},

		// Time only (today)
		{
			name:     "time only pm",
			input:    "3pm",
			wantTime: time.Date(2026, 1, 13, 15, 0, 0, 0, time.Local),
		},
		{
			name:     "time only with minutes",
			input:    "3:30pm",
			wantTime: time.Date(2026, 1, 13, 15, 30, 0, 0, time.Local),
		},
		{
			name:     "time only 24h",
			input:    "15:30",
			wantTime: time.Date(2026, 1, 13, 15, 30, 0, 0, time.Local),
		},

		// Date + time
		{
			name:     "jan date with time",
			input:    "Jan 15 3pm",
			wantTime: time.Date(2026, 1, 15, 15, 0, 0, 0, time.Local),
		},
		{
			name:     "january full with time",
			input:    "January 15 3:30pm",
			wantTime: time.Date(2026, 1, 15, 15, 30, 0, 0, time.Local),
		},

		// ISO format
		{
			name:     "iso date time",
			input:    "2026-01-15 15:30",
			wantTime: time.Date(2026, 1, 15, 15, 30, 0, 0, time.Local),
		},
		{
			name:     "iso date time T separator",
			input:    "2026-01-15T15:30",
			wantTime: time.Date(2026, 1, 15, 15, 30, 0, 0, time.Local),
		},

		// Full date with year
		{
			name:     "full date with year",
			input:    "Jan 15 2026 3pm",
			wantTime: time.Date(2026, 1, 15, 15, 0, 0, 0, time.Local),
		},

		// Errors
		{
			name:    "invalid input",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input, ref)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Parse(%q) expected error, got %v", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("Parse(%q) unexpected error: %v", tt.input, err)
				return
			}
			if !got.Equal(tt.wantTime) {
				t.Errorf("Parse(%q) = %v, want %v", tt.input, got, tt.wantTime)
			}
		})
	}
}
