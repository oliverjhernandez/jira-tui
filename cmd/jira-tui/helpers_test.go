package main

import (
	"testing"
	"time"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestFilterIssues(t *testing.T) {
	testIssues := []jira.Issue{
		{Key: "DEV-123", Status: "In Progress", Summary: "Fix loging bug"},
		{Key: "DEV-456", Status: "Done", Summary: "Add new feature"},
		{Key: "DEV-125", Status: "To Do", Summary: "Fix logout bug"},
	}

	tests := []struct {
		name     string
		filter   string
		expected int
	}{
		{
			name:     "filter by key",
			filter:   "DEV-123",
			expected: 1,
		},
		{
			name:     "filter by summary (case insensitive)",
			filter:   "BUG", // should match "bug" in two summaries
			expected: 2,
		},
		{
			name:     "filter by status",
			filter:   "done",
			expected: 1,
		},
		{
			name:     "no matches",
			filter:   "xyz",
			expected: 0,
		},
		{
			name:     "empty filter returns all",
			filter:   "",
			expected: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterIssues(testIssues, tt.filter)

			if len(result) != tt.expected {
				t.Errorf("filterIssues(%q) returned %d issues, expected %d", tt.filter, len(result), tt.expected)
			}
		})
	}
}

func TestTimeAgo(t *testing.T) {
	now := time.Now()
	jiraFormat := "2006-01-02T15:04:05.000-0700"

	tests := []struct {
		name     string
		date     string
		expected string
	}{
		{
			name:     "invalid date returns NA",
			date:     "invalid-date",
			expected: "NA",
		},
		{
			name:     "1 hour ago",
			date:     now.Add(-1 * time.Hour).Format(jiraFormat),
			expected: "1 hours ago",
		},
		{
			name:     "30 minutes ago",
			date:     now.Add(-30 * time.Minute).Format(jiraFormat),
			expected: "30 minutes ago",
		},
		{
			name:     "2 days ago",
			date:     now.Add(-48 * time.Hour).Format(jiraFormat),
			expected: "2 days ago",
		},
		{
			name:     "1 week ago",
			date:     now.Add(-7 * 24 * time.Hour).Format(jiraFormat),
			expected: "1 weeks ago",
		},
		{
			name:     "over a year shows date format",
			date:     now.Add(-400 * 24 * time.Hour).Format(jiraFormat),
			expected: now.Add(-400 * 24 * time.Hour).Local().Format("2006/01/02"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := timeAgo(tt.date)
			if result != tt.expected {
				t.Errorf("timeAgo(%q) = %q, expected %q", tt.date, result, tt.expected)
			}
		})
	}
}

func TestParseTimeToSeconds(t *testing.T) {
	tests := []struct {
		name      string
		time      string
		expected  int
		wantError bool
	}{

		{
			name:      "invalid time returns 0",
			time:      "4k",
			expected:  0,
			wantError: true,
		},
		{
			name:      "one hour",
			time:      "1h",
			expected:  3600,
			wantError: false,
		},
		{
			name:      "2 minutes",
			time:      "2m",
			expected:  120,
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseTimeToSeconds(tt.time)

			if tt.wantError {
				if err == nil {
					t.Errorf("parseTimeToSeconds(%q) expected error, returned nil", tt.time)
				}
				return
			}

			if err != nil {
				t.Errorf("parseTimeToSeconds(%q) unexpected error", tt.time)
				return
			}

			if result != tt.expected {
				t.Errorf("parseTimeToSeconds(%q) = %d, expected %d", tt.time, result, tt.expected)
			}
		})
	}
}
