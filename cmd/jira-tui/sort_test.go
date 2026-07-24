package main

import (
	"slices"
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestCompareIssuesOrder(t *testing.T) {
	issues := []jira.Issue{
		{Key: "A", Status: "To Do", Priority: jira.Priority{Name: "Low"}},
		{Key: "B", Status: "To Do", Priority: jira.Priority{Name: "Critica"}},
		{Key: "C", Status: "Done", Priority: jira.Priority{Name: "Critica"}}, // finished -> bottom
		{Key: "D", Status: "Trabajando", Priority: jira.Priority{Name: "High"}},
	}

	slices.SortFunc(issues, compareIssues)

	got := []string{issues[0].Key, issues[1].Key, issues[2].Key, issues[3].Key}
	want := []string{"B", "D", "A", "C"} // Critica, High, Low, then the finished one
	if !slices.Equal(got, want) {
		t.Errorf("order = %v, want %v", got, want)
	}
}
