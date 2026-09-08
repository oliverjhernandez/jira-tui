package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func modalModel(mode viewMode, base viewMode) model {
	m := newTabModel([]Tab{
		{id: 0, title: "Board", baseView: listView, board: boardState{jql: "a"}},
	}, 0)
	m.issues = []jira.Issue{{Key: "DEV-1", Status: "In Progress", Project: jira.Project{ID: "P"}}}
	m.sections = []Section{{Name: "All", Issues: m.issues}}
	m.selectedIssue = &m.sections[0].Issues[0]
	m.mode = mode
	m.baseView = base
	return m
}

func TestModalWithoutDataRecovers(t *testing.T) {
	modals := []viewMode{
		projectPickerView,
		savedBoardPickerView,
		issueSearchView,
		userSearchView,
		newIssueView,
		priorityView,
		estimateView,
		issueLinkView,
		summaryView,
		descriptionView,
		worklogView,
		cancelReasonView,
		blockReasonView,
		transitionWorklogView,
	}

	for _, mode := range modals {
		t.Run(mode.String(), func(t *testing.T) {
			t.Parallel()

			m := modalModel(mode, listView)
			var nm model
			assertNoPanic(t, mode.String(), func() {
				next, _ := m.Update(keyPress("j"))
				nm = next.(model)
			})
			if nm.mode != listView {
				t.Errorf("%v with no form data left mode = %v, want the base view", mode, nm.mode)
			}
		})
	}
}

func TestModalWithoutDataReturnsToDetail(t *testing.T) {
	t.Parallel()

	m := modalModel(userSearchView, detailView)
	m.activeIssue = &jira.Issue{Key: "DEV-2", ID: "2"}

	next, _ := m.Update(keyPress("j"))
	if got := next.(model).mode; got != detailView {
		t.Errorf("mode = %v, want detailView", got)
	}
}

func TestDetailLoadedKeepsPreviousModeNonModal(t *testing.T) {
	t.Parallel()

	m := modalModel(projectPickerView, listView)
	m.projectPickerData = NewProjectPickerFormData([]jira.Project{{ID: "P", Key: "PRJ", Name: "Proj"}}, 10)
	m.previousMode = listView

	next, _ := m.Update(issueDetailLoadedMsg{detail: &jira.Issue{Key: "DEV-1", ID: "1"}, tabID: 0})
	nm := next.(model)

	if nm.previousMode.isModal() {
		t.Errorf("previousMode = %v, a modal; restoring it drops the user into a form with no data", nm.previousMode)
	}
}

func TestAssignBeforeUsersLoadDoesNotPanic(t *testing.T) {
	t.Parallel()

	m := modalModel(listView, listView)
	m.usersCache = nil

	next, _ := m.Update(keyPress("a"))
	am := next.(model)
	if am.searchUserData != nil {
		t.Fatal("fixture assumes the picker opens with no data when users have not loaded")
	}

	var nm model
	assertNoPanic(t, "assign before users load", func() {
		next, _ = am.Update(keyPress("j"))
		nm = next.(model)
	})
	if nm.mode != listView {
		t.Errorf("mode = %v, want listView", nm.mode)
	}
}
