package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func sectionByCategory(sections []Section, key string) (Section, bool) {
	for _, s := range sections {
		if s.CategoryKey == key {
			return s, true
		}
	}
	return Section{}, false
}

func TestClassifyIssues(t *testing.T) {
	statuses := map[string][]jira.Status{
		"10": {
			{Name: "In Progress", StatusCategory: jira.StatusCategory{Key: "indeterminate"}},
			{Name: "To Do", StatusCategory: jira.StatusCategory{Key: "new"}},
			{Name: "Done", StatusCategory: jira.StatusCategory{Key: "done"}},
		},
	}

	issues := []jira.Issue{
		{Key: "DEV-1", Status: "In Progress", Project: jira.Project{ID: "10"}},
		{Key: "DEV-2", Status: "To Do", Project: jira.Project{ID: "10"}},
		{Key: "DEV-3", Status: "Done", Project: jira.Project{ID: "10"}},
		{Key: "DEV-4", Status: "Ready to Deploy", Project: jira.Project{ID: "10"}},
		{Key: "DEV-6", Status: "Validación", Project: jira.Project{ID: "10"}},
		{Key: "DEV-5", Status: "Totally Unknown", Project: jira.Project{ID: "10"}}, // unclassified
	}

	m := &model{}
	sections := m.classifyIssues(issues, statuses)

	checks := map[string]int{
		"indeterminate": 1,
		"new":           1,
		"done":          1,
		"transit":       1,
		"other":         1,
		"ready":         1,
	}
	for key, want := range checks {
		s, ok := sectionByCategory(sections, key)
		if !ok {
			t.Errorf("missing section with category %q", key)
			continue
		}
		if len(s.Issues) != want {
			t.Errorf("section %q has %d issues, want %d", key, len(s.Issues), want)
		}
	}
}

func TestClassifyIssuesReadyToDeployFirst(t *testing.T) {
	issues := []jira.Issue{
		{Key: "DEV-1", Status: "In Progress", StatusCategory: "indeterminate"},
		{Key: "DEV-2", Status: " ready to deploy ", StatusCategory: "indeterminate"},
	}

	m := &model{}
	sections := m.classifyIssues(issues, nil)

	if sections[0].CategoryKey != "ready" {
		t.Fatalf("first section = %q, want ready", sections[0].CategoryKey)
	}
	if len(sections[0].Issues) != 1 || sections[0].Issues[0].Key != "DEV-2" {
		t.Errorf("ready section issues = %+v, want [DEV-2]", sections[0].Issues)
	}
}

func TestClassifyIssuesSectionOrder(t *testing.T) {
	issues := []jira.Issue{
		{Key: "DEV-1", Status: "To Do", StatusCategory: "new"},
		{Key: "DEV-2", Status: "Validación", StatusCategory: "indeterminate"},
		{Key: "DEV-3", Status: "Ready to Deploy", StatusCategory: "indeterminate"},
		{Key: "DEV-4", Status: "In Progress", StatusCategory: "indeterminate"},
	}

	m := &model{}
	sections := m.classifyIssues(issues, nil)

	want := []string{"ready", "transit", "indeterminate", "new", "done"}
	if len(sections) != len(want) {
		t.Fatalf("got %d sections, want %d", len(sections), len(want))
	}
	for i, key := range want {
		if sections[i].CategoryKey != key {
			t.Errorf("section %d = %q, want %q", i, sections[i].CategoryKey, key)
		}
	}
}

func TestClassifyIssuesNoReadySectionWhenEmpty(t *testing.T) {
	issues := []jira.Issue{
		{Key: "DEV-1", Status: "In Progress", StatusCategory: "indeterminate"},
	}

	m := &model{}
	sections := m.classifyIssues(issues, nil)

	if _, ok := sectionByCategory(sections, "ready"); ok {
		t.Errorf("did not expect a 'ready' section when no issue is ready to deploy")
	}
}

func TestClassifyIssuesNoUnclassifiedSection(t *testing.T) {
	statuses := map[string][]jira.Status{
		"10": {
			{Name: "To Do", StatusCategory: jira.StatusCategory{Key: "new"}},
		},
	}
	issues := []jira.Issue{
		{Key: "DEV-1", Status: "To Do", Project: jira.Project{ID: "10"}},
	}

	m := &model{}
	sections := m.classifyIssues(issues, statuses)

	if _, ok := sectionByCategory(sections, "other"); ok {
		t.Errorf("did not expect an 'other' section when everything is classified")
	}
}

func TestCurrentIssue(t *testing.T) {
	m := model{
		sections: []Section{
			{Name: "In Progress", Issues: []jira.Issue{{Key: "DEV-1"}, {Key: "DEV-2"}}},
			{Name: "To Do", Issues: nil},
		},
	}

	tests := []struct {
		name       string
		sectionCur int
		cursor     int
		wantOK     bool
		wantKey    string
	}{
		{"valid first", 0, 0, true, "DEV-1"},
		{"valid second", 0, 1, true, "DEV-2"},
		{"cursor out of range", 0, 5, false, ""},
		{"negative cursor", 0, -1, false, ""},
		{"empty section", 1, 0, false, ""},
		{"section out of range", 9, 0, false, ""},
		{"negative section", -1, 0, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m.sectionCursor = tt.sectionCur
			m.cursor = tt.cursor
			issue, ok := m.currentIssue()
			if ok != tt.wantOK {
				t.Fatalf("currentIssue() ok = %v, want %v", ok, tt.wantOK)
			}
			if ok && issue.Key != tt.wantKey {
				t.Errorf("currentIssue() key = %q, want %q", issue.Key, tt.wantKey)
			}
		})
	}
}

func TestCurrentIssueUsesFilteredSections(t *testing.T) {
	m := model{
		sections: []Section{
			{Issues: []jira.Issue{{Key: "REAL-1"}}},
		},
		filteredSections: []Section{
			{Issues: []jira.Issue{{Key: "FILTERED-1"}}},
		},
	}
	issue, ok := m.currentIssue()
	if !ok || issue.Key != "FILTERED-1" {
		t.Errorf("currentIssue() should prefer filteredSections, got ok=%v key=%q", ok, issue.Key)
	}
}
