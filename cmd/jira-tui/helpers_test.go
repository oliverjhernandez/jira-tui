package main

import (
	"strings"
	"testing"
	"time"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
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
			name:     "1h ago",
			date:     now.Add(-1 * time.Hour).Format(jiraFormat),
			expected: "1h ago",
		},
		{
			name:     "30m ago",
			date:     now.Add(-30 * time.Minute).Format(jiraFormat),
			expected: "30m ago",
		},
		{
			name:     "2d ago",
			date:     now.Add(-48 * time.Hour).Format(jiraFormat),
			expected: "2d ago",
		},
		{
			name:     "1w ago",
			date:     now.Add(-7 * 24 * time.Hour).Format(jiraFormat),
			expected: "1w ago",
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
			result, err := parseStringToSeconds(tt.time)

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

func TestPlainClipboardText(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		markdown string
		want     []string
	}{
		{
			name:     "heading and paragraph",
			markdown: "# Release notes\n\nShipped the **new** board.\n",
			want:     []string{"# Release notes", "Shipped the new board."},
		},
		{
			name:     "bullet list keeps its items",
			markdown: "- **first** item\n- second item\n",
			want:     []string{"first item", "second item"},
		},
		{
			name:     "code block keeps its contents",
			markdown: "```\ngo test ./...\n```\n",
			want:     []string{"go test ./..."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			styled := jira.ExtractText(jira.MarkdownToADF(tt.markdown), 80)
			if !strings.Contains(styled, "\x1b") {
				t.Fatalf("renderer emitted no escape codes, test proves nothing:\n%q", styled)
			}

			got := plainClipboardText(styled)
			if strings.Contains(got, "\x1b") {
				t.Errorf("clipboard text still contains escape codes:\n%q", got)
			}
			for _, w := range tt.want {
				if !strings.Contains(got, w) {
					t.Errorf("clipboard text lost %q:\n%q", w, got)
				}
			}
		})
	}
}

func TestPlainClipboardTextStripsHyperlinks(t *testing.T) {
	t.Parallel()

	linked := ui.Osc8("https://example.com/x", ui.LinkStyle.Render("https://example.com/x"))
	got := plainClipboardText(linked)

	if got != "https://example.com/x" {
		t.Errorf("hyperlink yanked as %q, want the bare URL", got)
	}
}
