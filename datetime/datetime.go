package datetime

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Absolute date/time formats to try, in order of preference
var absoluteFormats = []string{
	// ISO 8601 variants
	"2006-01-02T15:04",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04",
	"2006-01-02 15:04:05",

	// Human-friendly with year
	"Jan 2 2006 3:04pm",
	"Jan 2 2006 3:04PM",
	"Jan 2 2006 3pm",
	"Jan 2 2006 3PM",
	"January 2 2006 3:04pm",
	"January 2 2006 3:04PM",
	"January 2 2006 3pm",
	"January 2 2006 3PM",

	// Human-friendly without year (will use current year)
	"Jan 2 3:04pm",
	"Jan 2 3:04PM",
	"Jan 2 3pm",
	"Jan 2 3PM",
	"January 2 3:04pm",
	"January 2 3:04PM",
	"January 2 3pm",
	"January 2 3PM",
}

// Time-only formats (will use today's date)
var timeOnlyFormats = []string{
	"3:04pm",
	"3:04PM",
	"3:04 pm",
	"3:04 PM",
	"3pm",
	"3PM",
	"3 pm",
	"3 PM",
	"15:04",
	"15:04:05",
}

// relativePattern matches strings like +2h, +30m, +1d, +1h30m
var relativePattern = regexp.MustCompile(`^\+(\d+[dhms])+$`)

// inPattern matches "in X days/hours/minutes"
var inPattern = regexp.MustCompile(`^in\s+(\d+)\s*(d|day|days|h|hour|hours|m|min|mins|minute|minutes)$`)

// weekdays for parsing
var weekdays = map[string]time.Weekday{
	"sunday": time.Sunday, "sun": time.Sunday,
	"monday": time.Monday, "mon": time.Monday,
	"tuesday": time.Tuesday, "tue": time.Tuesday, "tues": time.Tuesday,
	"wednesday": time.Wednesday, "wed": time.Wednesday,
	"thursday": time.Thursday, "thu": time.Thursday, "thurs": time.Thursday,
	"friday": time.Friday, "fri": time.Friday,
	"saturday": time.Saturday, "sat": time.Saturday,
}

// Parse attempts to parse a datetime string using multiple formats.
// relativeTo is used as the base time for relative times (e.g., +2h).
// Returns the parsed time or an error if no format matched.
func Parse(input string, relativeTo time.Time) (time.Time, error) {
	input = strings.TrimSpace(input)
	lower := strings.ToLower(input)

	// Try "tomorrow" variants
	if strings.HasPrefix(lower, "tomorrow") {
		return parseTomorrow(input, relativeTo)
	}

	// Try "in X days/hours" pattern
	if match := inPattern.FindStringSubmatch(lower); match != nil {
		return parseInDuration(match, relativeTo)
	}

	// Try weekday parsing
	if t, ok := parseWeekday(lower, relativeTo); ok {
		return t, nil
	}

	// Try relative time first
	if relativePattern.MatchString(input) {
		return parseRelative(input, relativeTo)
	}

	// Try each absolute format
	for _, format := range absoluteFormats {
		if t, err := time.ParseInLocation(format, input, time.Local); err == nil {
			// If no year was in the format, the parsed year will be 0
			// In that case, use the current year
			if t.Year() == 0 {
				t = t.AddDate(relativeTo.Year(), 0, 0)
			}
			return t, nil
		}
	}

	// Try time-only formats (use today's date)
	for _, format := range timeOnlyFormats {
		if t, err := time.ParseInLocation(format, input, time.Local); err == nil {
			// Combine today's date with the parsed time
			today := relativeTo
			return time.Date(today.Year(), today.Month(), today.Day(),
				t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse datetime: %q", input)
}

// parseRelative parses relative time strings like +2h, +30m, +1d, +1h30m
func parseRelative(input string, relativeTo time.Time) (time.Time, error) {
	// Remove the leading +
	input = input[1:]

	result := relativeTo
	current := ""

	for _, char := range input {
		if char >= '0' && char <= '9' {
			current += string(char)
		} else {
			if current == "" {
				return time.Time{}, fmt.Errorf("invalid relative time: missing number before %c", char)
			}

			num, err := strconv.Atoi(current)
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid number in relative time: %s", current)
			}

			switch char {
			case 'd':
				result = result.Add(time.Duration(num) * 24 * time.Hour)
			case 'h':
				result = result.Add(time.Duration(num) * time.Hour)
			case 'm':
				result = result.Add(time.Duration(num) * time.Minute)
			case 's':
				result = result.Add(time.Duration(num) * time.Second)
			default:
				return time.Time{}, fmt.Errorf("unknown time unit: %c", char)
			}

			current = ""
		}
	}

	return result, nil
}


// parseTomorrow handles "tomorrow" and "tomorrow 9am" style inputs
func parseTomorrow(input string, relativeTo time.Time) (time.Time, error) {
	tomorrow := relativeTo.AddDate(0, 0, 1)
	lower := strings.ToLower(input)

	// Just "tomorrow" - default to 9am
	if lower == "tomorrow" {
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
			9, 0, 0, 0, time.Local), nil
	}

	// "tomorrow <time>" - parse the time part
	timeStr := strings.TrimSpace(input[8:]) // len("tomorrow") = 8
	for _, format := range timeOnlyFormats {
		if t, err := time.ParseInLocation(format, timeStr, time.Local); err == nil {
			return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(),
				t.Hour(), t.Minute(), t.Second(), 0, time.Local), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse time in: %q", input)
}

// parseInDuration handles "in X days/hours/minutes" style inputs
func parseInDuration(match []string, relativeTo time.Time) (time.Time, error) {
	num, _ := strconv.Atoi(match[1])
	unit := match[2]

	switch unit[0] {
	case 'd':
		return relativeTo.AddDate(0, 0, num), nil
	case 'h':
		return relativeTo.Add(time.Duration(num) * time.Hour), nil
	case 'm':
		return relativeTo.Add(time.Duration(num) * time.Minute), nil
	}

	return time.Time{}, fmt.Errorf("unknown unit: %s", unit)
}

// parseWeekday handles "friday" or "friday 10am" style inputs
func parseWeekday(input string, relativeTo time.Time) (time.Time, bool) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return time.Time{}, false
	}

	targetDay, ok := weekdays[parts[0]]
	if !ok {
		return time.Time{}, false
	}

	// Calculate days until next occurrence
	currentDay := relativeTo.Weekday()
	daysUntil := int(targetDay) - int(currentDay)
	if daysUntil <= 0 {
		daysUntil += 7
	}

	targetDate := relativeTo.AddDate(0, 0, daysUntil)

	// Default to 9am
	hour, min := 9, 0

	// If there are more parts, they must form a valid time
	if len(parts) > 1 {
		timeStr := strings.Join(parts[1:], " ")
		parsed := false
		for _, format := range timeOnlyFormats {
			if t, err := time.ParseInLocation(format, timeStr, time.Local); err == nil {
				hour, min = t.Hour(), t.Minute()
				parsed = true
				break
			}
		}
		if !parsed {
			return time.Time{}, false
		}
	}

	return time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(),
		hour, min, 0, 0, time.Local), true
}
