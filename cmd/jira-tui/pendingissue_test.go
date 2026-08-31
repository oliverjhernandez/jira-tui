package main

import (
	"testing"

	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func pendingBoard(t *testing.T) model {
	t.Helper()

	m := newTabModel([]Tab{
		{id: 0, title: "Board", baseView: listView, board: boardState{jql: "a"}},
	}, 0)
	m.issues = []jira.Issue{
		{Key: "DEV-1", Status: "Done", Summary: "finished", Project: jira.Project{ID: "P"}},
		{Key: "DEV-2", Status: "In Progress", Summary: "active", Project: jira.Project{ID: "P"}},
	}
	m.sections = []Section{{Name: "All", Issues: m.issues}}
	m.sectionCursor = 0
	m.cursor = 0

	si, ok := m.currentIssue()
	if !ok {
		t.Fatal("fixture has no issue under the cursor")
	}
	m.selectedIssue = si
	return m
}

func TestPendingIssueSurvivesResort(t *testing.T) {
	t.Parallel()

	m := pendingBoard(t)
	wantKey := m.selectedIssue.Key

	m.setPendingIssue(m.selectedIssue)
	sortSectionsIssues(m.sections)

	if m.pendingIssue == nil {
		t.Fatal("pendingIssue went nil")
	}
	if m.pendingIssue.Key != wantKey {
		t.Errorf("pending issue retargeted to %q after re-sort, want %q", m.pendingIssue.Key, wantKey)
	}
	if m.sections[0].Issues[0].Key == wantKey {
		t.Fatal("fixture did not actually reorder; the test proves nothing")
	}
}

func TestSetPendingIssueSnapshots(t *testing.T) {
	t.Parallel()

	var m model

	m.setPendingIssue(nil)
	if m.pendingIssue != nil {
		t.Errorf("nil should clear pendingIssue, got %+v", m.pendingIssue)
	}

	src := &jira.Issue{Key: "DEV-9", Summary: "original"}
	m.setPendingIssue(src)
	if m.pendingIssue == src {
		t.Error("pendingIssue aliases the caller's issue instead of snapshotting it")
	}
	src.Summary = "mutated"
	if m.pendingIssue.Summary != "original" {
		t.Errorf("snapshot tracks later mutations: %q", m.pendingIssue.Summary)
	}
}

func completeAssign(t *testing.T, m model, userID string) model {
	t.Helper()

	m.searchUserData.ID = userID
	m.searchUserData.Form.State = huh.StateCompleted

	next, _ := m.updateSearchUserView(keyPress("enter"))
	return next.(model)
}

func TestAssignFromListDoesNotFetchDetail(t *testing.T) {
	t.Parallel()

	m := pendingBoard(t)
	m.usersCache = []jira.User{{ID: "u1", Name: "Someone"}}

	next, _ := m.Update(keyPress("a"))
	am := next.(model)
	if am.mode != userSearchView {
		t.Fatalf("`a` should open the user picker, mode = %v", am.mode)
	}
	if am.previousMode != listView {
		t.Fatalf("previousMode should record the list view, got %v", am.previousMode)
	}
	if am.focusedSection != metadataSection {
		t.Fatalf("fixture assumes the zero-value focusedSection, got %v", am.focusedSection)
	}
	if am.pendingIssue == nil || am.pendingIssue.Key != "DEV-1" {
		t.Fatalf("pending issue = %+v, want DEV-1", am.pendingIssue)
	}

	before := am.loadingCount
	done := completeAssign(t, am, "u1")

	if done.mode != listView {
		t.Errorf("assign should return to the list, mode = %v", done.mode)
	}
	if got := done.loadingCount - before; got != 1 {
		t.Errorf("assign from the list started %d requests, want 1 (board refresh only)", got)
	}
}

func TestAssignFromDetailRefreshesDetail(t *testing.T) {
	t.Parallel()

	m := pendingBoard(t)
	m.usersCache = []jira.User{{ID: "u1", Name: "Someone"}}
	m.mode = detailView
	m.activeIssue = &jira.Issue{Key: "DEV-2", Summary: "active"}
	m.focusedSection = metadataSection

	next, _ := m.Update(keyPress("a"))
	am := next.(model)
	if am.mode != userSearchView {
		t.Fatalf("`a` should open the user picker, mode = %v", am.mode)
	}

	before := am.loadingCount
	done := completeAssign(t, am, "u1")

	if got := done.loadingCount - before; got != 2 {
		t.Errorf("assign from the detail started %d requests, want 2 (detail + board)", got)
	}
}

func TestAssignFromDetailTargetsTheOpenIssue(t *testing.T) {
	t.Parallel()

	m := pendingBoard(t)
	m.usersCache = []jira.User{{ID: "u1", Name: "Someone"}}
	m.mode = detailView
	m.activeIssue = &jira.Issue{Key: "DEV-2", Summary: "the open one"}
	m.setPendingIssue(&jira.Issue{Key: "DEV-77", Summary: "stale"})

	next, _ := m.Update(keyPress("a"))
	am := next.(model)

	if am.pendingIssue == nil {
		t.Fatal("assign left pendingIssue nil")
	}
	if am.pendingIssue.Key != "DEV-2" {
		t.Errorf("assign targets %q, want the open issue DEV-2", am.pendingIssue.Key)
	}
	if am.userSelectionMode != assignUser {
		t.Errorf("userSelectionMode = %v, want assignUser", am.userSelectionMode)
	}
}
