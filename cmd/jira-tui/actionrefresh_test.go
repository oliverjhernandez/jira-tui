package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func listBoard() model {
	m := newTabModel([]Tab{
		{id: 0, title: "Board", baseView: listView, board: boardState{jql: "a"}},
	}, 0)
	m.issues = []jira.Issue{{Key: "DEV-1", Status: "In Progress", Project: jira.Project{ID: "P"}}}
	m.sections = []Section{{Name: "All", Issues: m.issues}}
	m.selectedIssue = &m.sections[0].Issues[0]
	m.mode = listView
	m.baseView = listView
	m.activeIssue = nil
	return m
}

func TestActionResultsFromListDoNotPanic(t *testing.T) {
	cases := []struct {
		name string
		msg  any
	}{
		{"transition", transitionPostedMsg{}},
		{"priority", priorityPostedMsg{}},
		{"estimate", estimatePostedMsg{}},
		{"issue link", issueLinkPostedMsg{}},
		{"description", updatedDescriptionMsg{}},
		{"summary", updatedSummaryMsg{}},
		{"comment posted", commentPostedMsg{}},
		{"comment updated", commentUpdatedMsg{}},
		{"comment deleted", commentDeletedMsg{}},
		{"worklog posted", workLogPostedMsg{}},
		{"worklog updated", workLogUpdatedMsg{}},
		{"worklog deleted", workLogDeletedMsg{}},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := listBoard()
			var nm model
			assertNoPanic(t, tt.name, func() {
				next, _ := m.Update(tt.msg)
				nm = next.(model)
			})
			if nm.mode == detailView {
				t.Errorf("%s from a list view switched to the detail view", tt.name)
			}
		})
	}
}

func TestTransitionFromDetailRefreshesDetail(t *testing.T) {
	t.Parallel()

	m := listBoard()
	m.mode = detailView
	m.baseView = detailView
	m.activeIssue = &jira.Issue{Key: "DEV-2", ID: "2"}
	m.focusedSection = metadataSection

	before := m.loadingCount
	next, _ := m.Update(transitionPostedMsg{})
	nm := next.(model)

	if nm.mode != detailView {
		t.Errorf("mode = %v, want detailView", nm.mode)
	}
	if got := nm.loadingCount - before; got != 0 {
		t.Errorf("loadingCount delta = %d, want 0 (one fetch queued, one msg consumed)", got)
	}
}

func TestDetailLoadedWithNilDetailDoesNotPanic(t *testing.T) {
	t.Parallel()

	m := listBoard()
	var nm model
	assertNoPanic(t, "nil detail", func() {
		next, _ := m.Update(issueDetailLoadedMsg{detail: nil, tabID: 0})
		nm = next.(model)
	})
	if nm.activeIssue != nil {
		t.Errorf("nil detail should not become the active issue: %+v", nm.activeIssue)
	}
}
