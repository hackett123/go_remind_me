package tui

import (
	"testing"
	"time"

	"go_remind/reminder"
)

// createTestModel creates a properly initialized Model for testing
// We pass nil for store since tests don't need persistence
func createTestModel(t *testing.T, reminders []*reminder.Reminder) *Model {
	t.Helper()
	m := New(reminders, nil, nil)
	return &m
}

func TestUpdateReminder(t *testing.T) {
	// Fixed reference time for consistent testing
	baseTime := time.Date(2026, 1, 13, 12, 0, 0, 0, time.Local)

	tests := []struct {
		name            string
		initialReminder *reminder.Reminder
		input           string
		wantErr         bool
		wantDesc        string
		wantTime        time.Time
		wantStatus      reminder.Status
	}{
		{
			name: "update description only with same time format",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime.Add(2 * time.Hour),
				Description: "Old description",
				Status:      reminder.Pending,
			},
			input:      "2026-01-13 14:00 New description",
			wantErr:    false,
			wantDesc:   "New description",
			wantTime:   baseTime.Add(2 * time.Hour),
			wantStatus: reminder.Pending,
		},
		{
			name: "update with relative time",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime,
				Description: "Old task",
				Status:      reminder.Triggered,
			},
			input:      "+1h Updated task",
			wantErr:    false,
			wantDesc:   "Updated task",
			wantStatus: reminder.Pending, // Future time should reset to pending
		},
		{
			name: "update with ISO datetime",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime,
				Description: "Meeting",
				Status:      reminder.Pending,
			},
			input:      "2026-01-15 15:30 Updated meeting",
			wantErr:    false,
			wantDesc:   "Updated meeting",
			wantTime:   time.Date(2026, 1, 15, 15, 30, 0, 0, time.Local),
			wantStatus: reminder.Pending,
		},
		{
			name: "update with natural language time",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime,
				Description: "Call mom",
				Status:      reminder.Pending,
			},
			input:    "tomorrow 3pm Call dad instead",
			wantErr:  false,
			wantDesc: "Call dad instead",
		},
		{
			name: "update multi-word description",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime,
				Description: "Short",
				Status:      reminder.Pending,
			},
			input:    "+2h This is a much longer description with many words",
			wantErr:  false,
			wantDesc: "This is a much longer description with many words",
		},
		{
			name: "acknowledged reminder stays acknowledged",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime.Add(-1 * time.Hour), // Past time
				Description: "Done task",
				Status:      reminder.Acknowledged,
			},
			input:      "2026-01-10 10:00 Still done task", // Past time
			wantErr:    false,
			wantDesc:   "Still done task",
			wantStatus: reminder.Acknowledged, // Should stay acknowledged
		},
		{
			name: "error on empty input",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime,
				Description: "Test",
				Status:      reminder.Pending,
			},
			input:   "",
			wantErr: true,
		},
		{
			name: "error on whitespace only",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime,
				Description: "Test",
				Status:      reminder.Pending,
			},
			input:   "   ",
			wantErr: true,
		},
		{
			name: "error on single word (no description)",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime,
				Description: "Test",
				Status:      reminder.Pending,
			},
			input:   "+1h",
			wantErr: true,
		},
		{
			name: "error on invalid datetime",
			initialReminder: &reminder.Reminder{
				DateTime:    baseTime,
				Description: "Test",
				Status:      reminder.Pending,
			},
			input:   "notadate Some description",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a properly initialized model with the reminder
			m := createTestModel(t, []*reminder.Reminder{tt.initialReminder})

			err := m.updateReminder(tt.initialReminder, tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("updateReminder() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("updateReminder() unexpected error: %v", err)
				return
			}

			// Check description
			if tt.wantDesc != "" && tt.initialReminder.Description != tt.wantDesc {
				t.Errorf("Description = %q, want %q", tt.initialReminder.Description, tt.wantDesc)
			}

			// Check time if specified
			if !tt.wantTime.IsZero() && !tt.initialReminder.DateTime.Equal(tt.wantTime) {
				t.Errorf("DateTime = %v, want %v", tt.initialReminder.DateTime, tt.wantTime)
			}

			// Check status if specified
			if tt.wantStatus != 0 && tt.initialReminder.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", tt.initialReminder.Status, tt.wantStatus)
			}
		})
	}
}

func TestUpdateReminderStatusTransitions(t *testing.T) {
	// Use a time we can control relative to "now"
	futureTime := time.Now().Add(1 * time.Hour)
	pastTime := time.Now().Add(-1 * time.Hour)

	tests := []struct {
		name          string
		initialStatus reminder.Status
		newTimeIsPast bool
		wantStatus    reminder.Status
	}{
		{
			name:          "pending stays pending when future",
			initialStatus: reminder.Pending,
			newTimeIsPast: false,
			wantStatus:    reminder.Pending,
		},
		{
			name:          "pending becomes triggered when past",
			initialStatus: reminder.Pending,
			newTimeIsPast: true,
			wantStatus:    reminder.Triggered,
		},
		{
			name:          "triggered becomes pending when future",
			initialStatus: reminder.Triggered,
			newTimeIsPast: false,
			wantStatus:    reminder.Pending,
		},
		{
			name:          "triggered stays triggered when past",
			initialStatus: reminder.Triggered,
			newTimeIsPast: true,
			wantStatus:    reminder.Triggered,
		},
		{
			name:          "acknowledged stays acknowledged when future",
			initialStatus: reminder.Acknowledged,
			newTimeIsPast: false,
			wantStatus:    reminder.Acknowledged,
		},
		{
			name:          "acknowledged stays acknowledged when past",
			initialStatus: reminder.Acknowledged,
			newTimeIsPast: true,
			wantStatus:    reminder.Acknowledged,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &reminder.Reminder{
				DateTime:    time.Now(),
				Description: "Test",
				Status:      tt.initialStatus,
			}

			m := createTestModel(t, []*reminder.Reminder{r})

			var input string
			if tt.newTimeIsPast {
				input = pastTime.Format("2006-01-02 15:04") + " Updated"
			} else {
				input = futureTime.Format("2006-01-02 15:04") + " Updated"
			}

			err := m.updateReminder(r, input)
			if err != nil {
				t.Fatalf("updateReminder() error: %v", err)
			}

			if r.Status != tt.wantStatus {
				t.Errorf("Status = %v, want %v", r.Status, tt.wantStatus)
			}
		})
	}
}

func TestEditPrefillFormat(t *testing.T) {
	// Test that the prefill format matches what we expect
	testTime := time.Date(2026, 1, 15, 14, 30, 0, 0, time.Local)
	r := &reminder.Reminder{
		DateTime:    testTime,
		Description: "Test reminder",
		Status:      reminder.Pending,
	}

	// This is the format used in updateNormalMode when pressing 'e'
	prefill := r.DateTime.Format("2006-01-02 15:04") + " " + r.Description
	expected := "2026-01-15 14:30 Test reminder"

	if prefill != expected {
		t.Errorf("Prefill format = %q, want %q", prefill, expected)
	}

	// Verify this format can be parsed back
	m := createTestModel(t, []*reminder.Reminder{r})

	err := m.updateReminder(r, prefill)
	if err != nil {
		t.Errorf("updateReminder() should parse prefill format: %v", err)
	}

	if r.Description != "Test reminder" {
		t.Errorf("Description after round-trip = %q, want %q", r.Description, "Test reminder")
	}

	if !r.DateTime.Equal(testTime) {
		t.Errorf("DateTime after round-trip = %v, want %v", r.DateTime, testTime)
	}
}
