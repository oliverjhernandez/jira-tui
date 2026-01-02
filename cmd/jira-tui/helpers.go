package main

import (
	"strings"

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

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func renderField(label, value string) string {
	return ui.DetailLabelStyle.Render(label+": ") + ui.DetailValueStyle.Render(value)
}

func issueMatchesFilter(issue jira.Issue, filter string) bool {
	filterLower := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(issue.Summary), filterLower) ||
		strings.Contains(strings.ToLower(issue.Key), filterLower) ||
		strings.Contains(strings.ToLower(issue.Status), filterLower)
}
