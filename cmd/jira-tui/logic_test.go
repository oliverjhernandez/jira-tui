package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestFormatSecondsToString(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{0, ""},
		{3600, "1h"},
		{5400, "1h 30m"},
		{120, "2m"},
		{59, ""},
	}
	for _, tt := range tests {
		if got := formatSecondsToString(tt.seconds); got != tt.want {
			t.Errorf("formatSecondsToString(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestExtractLoggedTime(t *testing.T) {
	worklogs := []jira.Worklog{
		{Time: 3600},
		{Time: 1800},
	}
	if got := extractLoggedTime(worklogs); got != "1h 30m" {
		t.Errorf("extractLoggedTime = %q, want 1h 30m", got)
	}
	if got := extractLoggedTime(nil); got != "" {
		t.Errorf("extractLoggedTime(nil) = %q, want empty", got)
	}
}

func TestSortIssuesByStatus(t *testing.T) {
	issues := []jira.Issue{
		{Key: "c", Status: "Done"},       // order 5
		{Key: "a", Status: "Trabajando"}, // order 1
		{Key: "b", Status: "To Do"},      // order 2
	}
	sortIssuesByStatus(issues)
	got := []string{issues[0].Key, issues[1].Key, issues[2].Key}
	want := []string{"a", "b", "c"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sortIssuesByStatus order = %v, want %v", got, want)
		}
	}
}

func TestSortSectionsIssuesStatusThenPriority(t *testing.T) {
	sections := []Section{
		{
			Issues: []jira.Issue{
				{Key: "low", Status: "To Do", Priority: jira.Priority{Name: "Low"}},
				{Key: "high", Status: "To Do", Priority: jira.Priority{Name: "High"}},
			},
		},
	}
	sortSectionsIssues(sections)
	if sections[0].Issues[0].Key != "high" {
		t.Errorf("expected higher priority first, got %q", sections[0].Issues[0].Key)
	}
}

func TestFindIndex(t *testing.T) {
	order := []focusedSection{metadataSection, descriptionSection, commentsSection}
	if got := findIndex(commentsSection, order); got != 2 {
		t.Errorf("findIndex(commentsSection) = %d, want 2", got)
	}
	// not present -> 0
	if got := findIndex(subTasksSection, order); got != 0 {
		t.Errorf("findIndex(missing) = %d, want 0", got)
	}
}

func TestPlainDescriptionToADF(t *testing.T) {
	doc := jira.MarkdownToADF("hello")
	if doc == nil {
		t.Fatal("MarkdownToADF returned nil")
	}
	if doc.Type != "doc" || doc.Version != 1 {
		t.Errorf("unexpected doc envelope: %+v", doc)
	}
	if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
		t.Fatalf("unexpected content: %+v", doc.Content)
	}
	para := doc.Content[0]
	if len(para.Content) != 1 || para.Content[0].Text != "hello" {
		t.Errorf("unexpected paragraph content: %+v", para.Content)
	}
}

func TestIsCancelTransition(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"Cancelar", true},
		{"Cancelado", true},
		{"CANCEL", true},
		{"Done", false},
		{"In Progress", false},
	}
	for _, tt := range tests {
		if got := isCancelTransition(jira.Transition{Name: tt.name}); got != tt.want {
			t.Errorf("isCancelTransition(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestFilterSections(t *testing.T) {
	sections := []Section{
		{
			Name: "In Progress",
			Issues: []jira.Issue{
				{Key: "DEV-1", Summary: "fix bug"},
				{Key: "DEV-2", Summary: "add feature"},
			},
		},
		{
			Name: "To Do",
			Issues: []jira.Issue{
				{Key: "DEV-3", Summary: "another bug"},
			},
		},
	}

	filtered := filterSections(sections, "bug", nil)
	if len(filtered) != 2 {
		t.Fatalf("filterSections should preserve section count, got %d", len(filtered))
	}
	if len(filtered[0].Issues) != 1 || filtered[0].Issues[0].Key != "DEV-1" {
		t.Errorf("first section filter wrong: %+v", filtered[0].Issues)
	}
	if len(filtered[1].Issues) != 1 || filtered[1].Issues[0].Key != "DEV-3" {
		t.Errorf("second section filter wrong: %+v", filtered[1].Issues)
	}
}

func TestIssueMatchesFilter(t *testing.T) {
	issue := jira.Issue{Key: "DEV-42", Summary: "Refactor parser", Status: "In Progress"}
	tags := []string{"needs-review", "urgent"}
	tests := []struct {
		name   string
		filter string
		tags   []string
		want   bool
	}{
		{name: "key", filter: "dev-42", want: true},
		{name: "summary", filter: "refactor", want: true},
		{name: "status", filter: "progress", want: true},
		{name: "no match", filter: "nonexistent", want: false},

		{name: "tag exact", filter: "#urgent", tags: tags, want: true},
		{name: "tag prefix as you type", filter: "#need", tags: tags, want: true},
		{name: "tag substring", filter: "#review", tags: tags, want: true},
		{name: "unknown tag", filter: "#nope", tags: tags, want: false},
		{name: "bare hash means has any tag", filter: "#", tags: tags, want: true},
		{name: "bare hash with no tags", filter: "#", want: false},
		{name: "tag filter ignores the summary", filter: "#refactor", tags: tags, want: false},
		{name: "plain filter ignores tags", filter: "urgent", tags: tags, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := issueMatchesFilter(issue, tt.filter, tt.tags); got != tt.want {
				t.Errorf("issueMatchesFilter(%q, tags=%v) = %v, want %v", tt.filter, tt.tags, got, tt.want)
			}
		})
	}
}

func TestFilterSectionsUsesTheTagLookup(t *testing.T) {
	sections := []Section{{
		Name: "In Progress",
		Issues: []jira.Issue{
			{Key: "DEV-1", Summary: "alpha"},
			{Key: "DEV-2", Summary: "bravo"},
		},
	}}
	tagsOf := func(key string) []string {
		if key == "DEV-2" {
			return []string{"urgent"}
		}
		return nil
	}

	filtered := filterSections(sections, "#urgent", tagsOf)
	if len(filtered) != 1 {
		t.Fatalf("filterSections should preserve section count, got %d", len(filtered))
	}
	if len(filtered[0].Issues) != 1 || filtered[0].Issues[0].Key != "DEV-2" {
		t.Errorf("a #tag filter should keep only the tagged issue, got %+v", filtered[0].Issues)
	}

	if got := filterSections(sections, "#urgent", nil); len(got[0].Issues) != 0 {
		t.Errorf("with no tag lookup a #tag filter should match nothing, got %+v", got[0].Issues)
	}
}
