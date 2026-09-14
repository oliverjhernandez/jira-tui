package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func filteringModel(t *testing.T, filter string, issues ...jira.Issue) model {
	t.Helper()
	m := newTabModel([]Tab{{id: 0, baseView: listView}}, 0)
	m.projects = []jira.Project{{ID: "10", Key: "DEV"}}
	m.issues = issues
	m.sections = m.sectionsFor(m.issues)
	if filter != "" {
		m.filtering = true
		m.textInput.SetValue(filter)
	}
	m.rebuildFilteredSections()
	return m
}

func issue(key, summary string) jira.Issue {
	return jira.Issue{Key: key, Summary: summary, Status: "To Do"}
}

func TestSelectIssueByKeyMatchesTheRenderedRow(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "bravo",
		issue("DEV-1", "alpha"),
		issue("DEV-2", "bravo"),
		issue("DEV-3", "bravo"),
		issue("DEV-4", "bravo"),
	)

	m.selectIssueByKey("DEV-3")

	highlighted, ok := m.currentIssue()
	if !ok {
		t.Fatalf("cursor out of range: sectionCursor=%d cursor=%d", m.sectionCursor, m.cursor)
	}
	if m.selectedIssue == nil || m.selectedIssue.Key != "DEV-3" {
		t.Fatalf("selectedIssue = %v, want DEV-3", m.selectedIssue)
	}
	if highlighted.Key != m.selectedIssue.Key {
		t.Errorf("highlighted row %s but selection is %s: acting on the cursor would hit the wrong issue",
			highlighted.Key, m.selectedIssue.Key)
	}
}

func TestSelectIssueByKeyUnfilteredStillWorks(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "",
		issue("DEV-1", "alpha"),
		issue("DEV-2", "bravo"),
		issue("DEV-3", "charlie"),
	)

	m.selectIssueByKey("DEV-3")

	got, ok := m.currentIssue()
	if !ok || got.Key != "DEV-3" || m.selectedIssue.Key != "DEV-3" {
		t.Errorf("selection = %v / row = %v, want DEV-3", m.selectedIssue, got)
	}
}

func TestSelectIssueByKeyFallsBackToFirstVisibleIssue(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "bravo",
		issue("DEV-1", "alpha"),
		issue("DEV-2", "bravo"),
		issue("DEV-3", "bravo"),
	)

	// DEV-9 is not on the board at all (e.g. transitioned out of the JQL).
	m.selectIssueByKey("DEV-9")

	got, ok := m.currentIssue()
	if !ok {
		t.Fatal("cursor out of range after falling back")
	}
	if got.Key != "DEV-2" {
		t.Errorf("fell back to %s, want the first *visible* issue DEV-2", got.Key)
	}
	if m.selectedIssue.Key != got.Key {
		t.Errorf("selection %s disagrees with row %s", m.selectedIssue.Key, got.Key)
	}
}

func TestRefreshRebuildsTheFilteredView(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "bravo",
		issue("DEV-1", "alpha"),
		issue("DEV-2", "bravo"),
	)
	m.selectIssueByKey("DEV-2")

	// A refresh brings in a new issue that matches the active filter.
	m.issues = []jira.Issue{
		issue("DEV-1", "alpha"),
		issue("DEV-2", "bravo"),
		issue("DEV-5", "bravo"),
	}
	next, _ := m.Update(statusesLoadedMsg{statuses: map[string][]jira.Status{}, tabID: 0})
	am := next.(model)

	var visible []string
	for _, s := range am.navSections() {
		for _, i := range s.Issues {
			visible = append(visible, i.Key)
		}
	}

	if len(visible) != 2 {
		t.Errorf("filtered view shows %v, want the two bravo issues (stale filter not rebuilt)", visible)
	}
	for _, k := range visible {
		if k == "DEV-1" {
			t.Errorf("filtered view leaked the non-matching DEV-1: %v", visible)
		}
	}
	if am.selectedIssue == nil || am.selectedIssue.Key != "DEV-2" {
		t.Errorf("selection = %v, want DEV-2 preserved across the refresh", am.selectedIssue)
	}
	if row, ok := am.currentIssue(); !ok || row.Key != "DEV-2" {
		t.Errorf("highlighted row = %v (ok=%v), want DEV-2", row, ok)
	}
}

func TestRefreshKeepsSelectionWhenUnfiltered(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "",
		issue("DEV-1", "alpha"),
		issue("DEV-2", "bravo"),
	)
	m.selectIssueByKey("DEV-2")

	// New issue arrives ahead of the selection in the list.
	m.issues = []jira.Issue{
		issue("DEV-0", "zulu"),
		issue("DEV-1", "alpha"),
		issue("DEV-2", "bravo"),
	}
	next, _ := m.Update(statusesLoadedMsg{statuses: map[string][]jira.Status{}, tabID: 0})
	am := next.(model)

	if am.selectedIssue == nil || am.selectedIssue.Key != "DEV-2" {
		t.Errorf("selection = %v, want DEV-2 to follow its issue", am.selectedIssue)
	}
	if row, ok := am.currentIssue(); !ok || row.Key != "DEV-2" {
		t.Errorf("highlighted row = %v (ok=%v), want DEV-2", row, ok)
	}
	if am.filteredSections != nil {
		t.Error("filteredSections should stay nil when no filter is active")
	}
}
