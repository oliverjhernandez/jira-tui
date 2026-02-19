package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

var priorityOrder = map[string]int{
	"Crítica": 1,
	"Highest": 2,
	"High":    3,
	"Medium":  4,
	"Low":     5,
	"Lowest":  6,
}

func filterIssues(issues []jira.Issue, filter string) []jira.Issue {
	var filtered []jira.Issue

	for _, i := range issues {
		if issueMatchesFilter(i, filter) {
			filtered = append(filtered, i)
		}
	}

	return filtered
}

func filterSections(sections []Section, filter string) []Section {
	var filteredSections []Section

	for _, s := range sections {
		var filteredIssues []*jira.Issue
		for _, i := range s.Issues {
			if issueMatchesFilter(*i, filter) {
				filteredIssues = append(filteredIssues, i)
			}
		}
		if len(filteredIssues) > 0 {
			s.Issues = filteredIssues
		} else {
			s.Issues = nil
		}

		filteredSections = append(filteredSections, s)
	}
	return filteredSections
}

func truncateLongString(s string, max int) string {
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}

func issueMatchesFilter(issue jira.Issue, filter string) bool {
	filterLower := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(issue.Summary), filterLower) ||
		strings.Contains(strings.ToLower(issue.Key), filterLower)
}

func timeAgo(date string) string {
	formats := []string{
		"2006-01-02T15:04:05.000-0700", // Original format
		time.RFC3339,                   // "2025-10-23T18:02:52Z"
	}
	defaultDatetime := "NA" // NOTE: to define
	now := time.Now()

	var datetime time.Time
	var err error
	for _, f := range formats {
		datetime, err = time.Parse(f, date)
		if err == nil {
			break
		}
	}
	if err != nil {
		log.Printf("error parsing comment date: %s ", err.Error())
		return defaultDatetime
	}

	diff := now.Sub(datetime)
	diffHours := int(diff.Hours())
	diffMinutes := int(diff.Minutes())
	oneDay := 24
	oneWeek := oneDay * 7
	oneMonth := oneDay * 30
	oneYear := oneMonth * 12

	if diffHours >= oneYear {
		return datetime.Local().Format("2006/01/02")
	} else if diffHours >= oneMonth {
		return strconv.Itoa(diffHours/oneMonth) + " months ago"
	} else if diffHours >= oneWeek {
		return strconv.Itoa(diffHours/oneWeek) + " weeks ago"
	} else if diffHours >= oneDay {
		return strconv.Itoa(diffHours/oneDay) + " days ago"
	} else if diffHours >= 1 {
		return strconv.Itoa(diffHours) + " hours ago"
	} else {
		return strconv.Itoa(diffMinutes) + " minutes ago"
	}
}

func extractLoggedTime(worklogs []jira.WorkLog) string {
	logged := 0

	if len(worklogs) > 0 {
		for _, wl := range worklogs {
			logged += wl.Time
		}
	}

	loggedHours := logged / 60 / 60
	loggedStr := strconv.Itoa(loggedHours)

	return loggedStr + "h"
}

func parseTimeToSeconds(input string) (int, error) {
	input = strings.TrimSpace(strings.ToLower(input))

	totalSeconds := 0

	if strings.Contains(input, "h") {
		parts := strings.Split(input, "h")
		hours, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid hours: %s", parts[0])
		}
		totalSeconds += int(hours * 3600)
		input = strings.TrimSpace(parts[1])
	}

	if strings.Contains(input, "m") {
		parts := strings.Split(input, "m")
		minutes, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		if err != nil {
			return 0, fmt.Errorf("invalid minutes: %s", parts[0])
		}
		totalSeconds += int(minutes * 60)
	}

	if totalSeconds == 0 {
		return 0, fmt.Errorf("could not parse time: %s", input)
	}

	return totalSeconds, nil
}

func (m model) getAbsoluteCursorLine() int {
	lines := 0

	headerLines := 3
	sectionSpacing := 1

	for i := 0; i < m.sectionCursor; i++ {
		lines += headerLines + sectionSpacing
		lines += len(m.sections[i].Issues)
	}

	lines += 2
	lines += m.cursor + 1

	return lines
}

func (m model) getCommentCursorLine() int {
	lines := 0
	width := m.detailLayout.leftColumnWidth

	for i := 0; i < m.commentsCursor; i++ {
		c := m.issueDetail.Comments[i]

		lines += 1

		bodyText := jira.ExtractText(c.Body, width-4)
		wrappedBody := ui.CommentBodyStyle.Width(width - 4).Render(bodyText)
		lines += lipgloss.Height(wrappedBody)

		lines += 2
	}

	return lines
}

func sortSectionsByPriority(sections []Section) {
	for si := range sections {
		sort.Slice(sections[si].Issues, func(i, j int) bool {
			return priorityOrder[sections[si].Issues[i].Priority] < priorityOrder[sections[si].Issues[j].Priority]
		})
	}
}

func findIndex(section focusedSection, order []focusedSection) int {
	for i, s := range order {
		if s == section {
			return i
		}
	}
	return 0
}
