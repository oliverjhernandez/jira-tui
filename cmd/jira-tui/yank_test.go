package main

import (
	"strings"
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func paragraphDoc(paragraphs ...string) *jira.ContentDoc {
	doc := &jira.ContentDoc{Type: "doc", Version: 1}
	for _, p := range paragraphs {
		doc.Content = append(doc.Content, jira.ContentNode{
			Type:    "paragraph",
			Content: []jira.ContentNode{{Type: "text", Text: p}},
		})
	}
	return doc
}

func TestIssueYankBlock(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   jira.Issue
		want string
	}{
		{
			name: "key, summary and description",
			in: jira.Issue{
				Key:         "DEV-1",
				Summary:     "Fix the login timeout",
				Description: paragraphDoc("Sessions drop after 30s."),
			},
			want: "DEV-1 // Fix the login timeout\nSessions drop after 30s.",
		},
		{
			name: "no description leaves no trailing blank line",
			in:   jira.Issue{Key: "DEV-2", Summary: "Just a title"},
			want: "DEV-2 // Just a title",
		},
		{
			name: "an empty description document is the same as none",
			in: jira.Issue{
				Key:         "DEV-3",
				Summary:     "Empty doc",
				Description: &jira.ContentDoc{Type: "doc", Version: 1},
			},
			want: "DEV-3 // Empty doc",
		},
		{
			name: "multiple paragraphs are kept",
			in: jira.Issue{
				Key:         "DEV-4",
				Summary:     "Two paragraphs",
				Description: paragraphDoc("First.", "Second."),
			},
			want: "DEV-4 // Two paragraphs\nFirst.\n\nSecond.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := issueYankBlock(tt.in); got != tt.want {
				t.Errorf("issueYankBlock() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestIssueYankBlockHeaderIsAlwaysTheFirstLine(t *testing.T) {
	t.Parallel()

	block := issueYankBlock(jira.Issue{
		Key:         "DEV-9",
		Summary:     "Multi line body",
		Description: paragraphDoc("line one", "line two"),
	})

	first, _, _ := strings.Cut(block, "\n")
	if first != "DEV-9 // Multi line body" {
		t.Errorf("first line = %q, want the KEY // Summary header", first)
	}
}

// TestIssueYankBlockSurvivesAnEmptyIssue guards the list path, where an issue
// can be a zero value if the cursor and sections ever disagree.
func TestIssueYankBlockSurvivesAnEmptyIssue(t *testing.T) {
	t.Parallel()

	assertNoPanic(t, "yank block of a zero issue", func() {
		if got := issueYankBlock(jira.Issue{}); got != " // " {
			t.Errorf("issueYankBlock(zero) = %q", got)
		}
	})
}
