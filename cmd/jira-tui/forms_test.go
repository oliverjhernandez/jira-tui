package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestNewTransitionFormDataStoresTransitions(t *testing.T) {
	transitions := []jira.Transition{
		{ID: "1", Name: "To Do"},
		{ID: "2", Name: "In Progress"},
		{ID: "3", Name: "Done"},
	}

	fd := NewTransitionFormData(transitions)
	if fd == nil {
		t.Fatal("NewTransitionFormData returned nil")
	}
	if fd.SelectedIndex != 0 {
		t.Errorf("default SelectedIndex = %d, want 0", fd.SelectedIndex)
	}
	if len(fd.Transitions) != len(transitions) {
		t.Fatalf("stored %d transitions, want %d", len(fd.Transitions), len(transitions))
	}
	for i := range transitions {
		if fd.Transitions[i].ID != transitions[i].ID {
			t.Errorf("Transitions[%d].ID = %q, want %q", i, fd.Transitions[i].ID, transitions[i].ID)
		}
	}
}

func TestNewTransitionFormDataEmpty(t *testing.T) {
	fd := NewTransitionFormData(nil)
	if fd == nil {
		t.Fatal("NewTransitionFormData(nil) returned nil")
	}
	if len(fd.Transitions) != 0 {
		t.Errorf("expected no transitions, got %d", len(fd.Transitions))
	}
}

func TestNewSearchUserFormData(t *testing.T) {
	users := []jira.User{
		{ID: "a1", Name: "Alice"},
		{ID: "b2", Name: "Bob"},
	}
	fd := NewSearchUserFormData(users)
	if fd == nil || fd.Form == nil {
		t.Fatal("NewSearchUserFormData returned nil form")
	}
}

func TestNewPriorityFormDataPreselectsCurrent(t *testing.T) {
	priorities := []jira.Priority{
		{ID: "1", Name: "High"},
		{ID: "2", Name: "Low"},
	}
	fd := NewPriorityFormData(priorities, "Low")
	if fd == nil || fd.Form == nil {
		t.Fatal("NewPriorityFormData returned nil form")
	}
	if fd.SelectedPriority != "Low" {
		t.Errorf("SelectedPriority = %q, want Low", fd.SelectedPriority)
	}
}

func TestNewWorklogFormFormatsTime(t *testing.T) {
	m := model{}
	wl := &jira.Worklog{ID: 7, Time: 5400, StartDate: "2024-01-01", Description: "work"}
	fd := m.NewWorklogForm(wl, 40)
	if fd == nil || fd.Form == nil {
		t.Fatal("NewWorklogForm returned nil form")
	}
	if fd.ID != 7 {
		t.Errorf("ID = %d, want 7", fd.ID)
	}
	if fd.Time != "1h 30m" {
		t.Errorf("Time = %q, want 1h 30m", fd.Time)
	}
}

func TestNewIssueFormNilMyself(t *testing.T) {
	// Should not panic when myself has not loaded yet.
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("NewIssueForm panicked with nil myself: %v", r)
		}
	}()
	m := model{}
	fd := m.NewIssueForm(&NewIssueFormData{})
	if fd == nil || fd.Form == nil {
		t.Fatal("NewIssueForm returned nil form")
	}
}

func TestNewIssueLinkForm(t *testing.T) {
	fd := NewIssueLinkForm(40)
	if fd == nil || fd.Form == nil {
		t.Fatal("NewIssueLinkForm returned nil form")
	}
}
