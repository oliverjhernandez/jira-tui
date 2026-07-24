package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestIsEpic(t *testing.T) {
	if !isEpic(jira.Issue{Type: "Epic"}) {
		t.Error("Epic should be an epic")
	}
	if isEpic(jira.Issue{Type: "Task"}) {
		t.Error("Task should not be an epic")
	}
}

func TestGroupByEpic(t *testing.T) {
	issues := []jira.Issue{
		{Key: "P-10", Type: "Epic", Summary: "Billing"},
		{Key: "P-11", Type: "Task", Parent: &jira.Parent{Key: "P-10"}},
		{Key: "P-20", Type: "Epic", Summary: "Onboarding"},
		{Key: "P-12", Type: "Task", Parent: &jira.Parent{Key: "P-10"}},
		{Key: "P-30", Type: "Task"},                                 // no epic
		{Key: "P-31", Type: "Task", Parent: &jira.Parent{Key: "X"}}, // epic not present
	}

	sections := groupByEpic(issues)

	// P-10, P-20 epics in input order, then a "No epic" section.
	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d: %+v", len(sections), sections)
	}
	if sections[0].Epic == nil || sections[0].Epic.Key != "P-10" {
		t.Errorf("section 0 should be epic P-10, got %+v", sections[0].Epic)
	}
	if len(sections[0].Issues) != 2 {
		t.Errorf("epic P-10 should have 2 tasks, got %d", len(sections[0].Issues))
	}
	if sections[0].Issues[0].Key != "P-11" || sections[0].Issues[1].Key != "P-12" {
		t.Errorf("P-10 tasks out of order: %+v", sections[0].Issues)
	}
	if sections[1].Epic == nil || sections[1].Epic.Key != "P-20" {
		t.Errorf("section 1 should be epic P-20, got %+v", sections[1].Epic)
	}
	if len(sections[1].Issues) != 0 {
		t.Errorf("epic P-20 should have no tasks, got %d", len(sections[1].Issues))
	}
	last := sections[2]
	if last.Epic != nil || last.Name != "No epic" || len(last.Issues) != 2 {
		t.Errorf("last section should be 'No epic' with 2 tasks (orphan + missing-epic), got %+v", last)
	}
}
