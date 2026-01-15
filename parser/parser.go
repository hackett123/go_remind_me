package parser

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"go_remind/datetime"
	"go_remind/reminder"
)

// Pattern matches [remind_me <content>]
var remindPattern = regexp.MustCompile(`\[remind_me\s+([^\]]+)\]`)

// Pattern matches #tag tokens (word characters after #, must be preceded by start or whitespace)
var tagPattern = regexp.MustCompile(`(?:^|\s)#(\w+)`)

// ParseFile reads a markdown file and extracts all reminders.
// relativeTo is used as the base time for relative datetime parsing.
func ParseFile(filepath string, relativeTo time.Time) ([]*reminder.Reminder, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var reminders []*reminder.Reminder
	scanner := bufio.NewScanner(file)
	lineNumber := 0

	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()

		matches := remindPattern.FindAllStringSubmatch(line, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue
			}

			content := strings.TrimSpace(match[1])
			r, err := parseReminderContent(content, relativeTo)
			if err != nil {
				// Skip invalid reminders but could log warning
				continue
			}

			r.SourceFile = filepath
			r.LineNumber = lineNumber
			reminders = append(reminders, r)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return reminders, nil
}

// ExtractTags extracts #tag tokens from text and returns the cleaned text and tags.
// Tags must be preceded by whitespace or be at the start of the string.
func ExtractTags(text string) (cleanText string, tags []string) {
	matches := tagPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) >= 2 {
			tags = append(tags, match[1])
		}
	}

	// Remove tag tokens from text (including the # prefix)
	cleanText = tagPattern.ReplaceAllString(text, "")
	cleanText = strings.TrimSpace(cleanText)
	// Clean up any double spaces left behind
	cleanText = strings.Join(strings.Fields(cleanText), " ")

	return cleanText, tags
}

// parseReminderContent parses the content inside [remind_me <content>]
// It tries progressively longer prefixes as the datetime until one parses successfully.
// The remainder becomes the description.
func parseReminderContent(content string, relativeTo time.Time) (*reminder.Reminder, error) {
	words := strings.Fields(content)
	if len(words) < 2 {
		return nil, fmt.Errorf("reminder must have both datetime and description")
	}

	// Try parsing from longest to shortest datetime prefix
	// This ensures "friday 10am" is tried before "friday"
	for numDateWords := len(words) - 1; numDateWords >= 1; numDateWords-- {
		dateStr := strings.Join(words[:numDateWords], " ")
		descStr := strings.Join(words[numDateWords:], " ")

		parsedTime, err := datetime.Parse(dateStr, relativeTo)
		if err == nil {
			// Extract tags from description
			cleanDesc, tags := ExtractTags(descStr)
			return &reminder.Reminder{
				DateTime:    parsedTime,
				Description: cleanDesc,
				Tags:        tags,
				Status:      reminder.Pending,
			}, nil
		}
	}

	return nil, fmt.Errorf("could not parse datetime from: %s", content)
}
