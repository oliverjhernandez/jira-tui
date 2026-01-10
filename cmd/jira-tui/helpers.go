package main

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func filterIssues(issues []jira.Issue, filter string) []jira.Issue {
	var filtered []jira.Issue

	for _, i := range issues {
		if issueMatchesFilter(i, filter) {
			filtered = append(filtered, i)
		}
	}

	return filtered
}

func renderStatusBadge(status string) string {
	if strings.ToLower(status) == "selected for development" {
		status = "To Do"
	}

	statusLower := strings.ToLower(status)

	if strings.Contains(statusLower, "trabajando") {
		return ui.StatusInProgressStyle.Render(status)
	} else if strings.Contains(statusLower, "done") {
		return ui.StatusDoneStyle.Render(status)
	} else if strings.Contains(statusLower, "backlog") || strings.Contains(statusLower, "to do") {
		return ui.StatusToDoStyle.Render(status)
	}

	return ui.StatusDefaultStyle.Render(status)
}

func truncateLongString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func renderField(label, value string) string {
	return ui.DetailFieldStyle.Render(
		ui.DetailLabelStyle.Render(label+": "),
		ui.DetailValueStyle.Render(value),
	)
}

func issueMatchesFilter(issue jira.Issue, filter string) bool {
	filterLower := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(issue.Summary), filterLower) ||
		strings.Contains(strings.ToLower(issue.Key), filterLower) ||
		strings.Contains(strings.ToLower(issue.Status), filterLower)
}

func timeAgo(date string) string {
	jiraFormat := "2006-01-02T15:04:05.000-0700"
	defaultDatetime := "NA" // NOTE: to define
	now := time.Now()

	datetime, err := time.Parse(jiraFormat, date)
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
