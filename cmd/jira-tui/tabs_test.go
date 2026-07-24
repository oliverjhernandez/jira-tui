package main

import (
	"testing"

	"charm.land/bubbles/v2/textinput"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func newTabModel(tabs []Tab, active int) model {
	return model{
		mode:         listView,
		baseView:     listView,
		activeTab:    active,
		nextTabID:    len(tabs),
		tabs:         tabs,
		windowWidth:  100,
		windowHeight: 40,
		textInput:    textinput.New(),
		statuses: map[string][]jira.Status{
			"P": {{Name: "In Progress", StatusCategory: jira.StatusCategory{Key: "indeterminate"}}},
		},
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	m := newTabModel([]Tab{
		{id: 0, title: "A", baseView: listView, board: boardState{jql: "jqlA"}},
		{id: 1, title: "B", baseView: listView, board: boardState{jql: "jqlB"}},
	}, 0)

	// Populate live (flat) state for the active tab (id 0).
	m.issues = []jira.Issue{
		{Key: "a", Status: "In Progress", Project: jira.Project{ID: "P"}},
		{Key: "b", Status: "In Progress", Project: jira.Project{ID: "P"}},
	}
	m.cursor = 1
	m.sectionCursor = 0
	m.activeIssue = &jira.Issue{Key: "DEV-99"}
	m.focusedSection = commentsSection
	m.commentsCursor = 2

	m.saveActiveTab()

	// Corrupt the live state.
	m.issues = nil
	m.cursor = 99
	m.sectionCursor = 99
	m.activeIssue = nil
	m.focusedSection = metadataSection
	m.commentsCursor = 0

	m.loadActiveTab()

	if len(m.issues) != 2 {
		t.Fatalf("issues not restored: %d", len(m.issues))
	}
	if m.cursor != 1 || m.sectionCursor != 0 {
		t.Errorf("cursor/sectionCursor not restored: %d/%d", m.cursor, m.sectionCursor)
	}
	if m.activeIssue == nil || m.activeIssue.Key != "DEV-99" {
		t.Errorf("activeIssue not restored: %+v", m.activeIssue)
	}
	if m.focusedSection != commentsSection || m.commentsCursor != 2 {
		t.Errorf("detail cursors not restored: %v/%d", m.focusedSection, m.commentsCursor)
	}
	// selectedIssue is recomputed from the cursor, not stored.
	if m.selectedIssue == nil || m.selectedIssue.Key != "b" {
		t.Errorf("selectedIssue not recomputed correctly: %+v", m.selectedIssue)
	}
}

func TestSwitchWrap(t *testing.T) {
	m := newTabModel([]Tab{
		{id: 0, baseView: listView, board: boardState{jql: "a"}},
		{id: 1, baseView: listView, board: boardState{jql: "b"}},
		{id: 2, baseView: listView, board: boardState{jql: "c"}},
	}, 0)

	next, _ := m.switchTab(-1)
	if got := next.(model).activeTab; got != 2 {
		t.Errorf("switchTab(-1) from 0 = %d, want 2", got)
	}

	m.activeTab = 2
	next, _ = m.switchTab(+1)
	if got := next.(model).activeTab; got != 0 {
		t.Errorf("switchTab(+1) from 2 = %d, want 0", got)
	}

	single := newTabModel([]Tab{{id: 0, baseView: listView}}, 0)
	next, _ = single.switchTab(+1)
	if got := next.(model).activeTab; got != 0 {
		t.Errorf("switchTab with one tab changed activeTab to %d", got)
	}
}

func TestCloseSemantics(t *testing.T) {
	// Cannot close the last tab.
	single := newTabModel([]Tab{{id: 0, baseView: listView}}, 0)
	next, _ := single.closeActiveTab()
	if len(next.(model).tabs) != 1 {
		t.Errorf("closing the last tab should be refused")
	}

	// Close active middle tab -> neighbor becomes active.
	m := newTabModel([]Tab{
		{id: 0, baseView: listView, board: boardState{jql: "a"}},
		{id: 1, baseView: listView, board: boardState{jql: "b"}},
		{id: 2, baseView: listView, board: boardState{jql: "c"}},
	}, 1)
	next, _ = m.closeActiveTab()
	nm := next.(model)
	if len(nm.tabs) != 2 {
		t.Fatalf("expected 2 tabs after close, got %d", len(nm.tabs))
	}
	if nm.tabs[nm.activeTab].id != 2 {
		t.Errorf("after closing middle, active id = %d, want 2", nm.tabs[nm.activeTab].id)
	}

	// Close active last tab -> previous becomes active.
	m2 := newTabModel([]Tab{
		{id: 0, baseView: listView, board: boardState{jql: "a"}},
		{id: 1, baseView: listView, board: boardState{jql: "b"}},
	}, 1)
	next, _ = m2.closeActiveTab()
	nm2 := next.(model)
	if nm2.activeTab != 0 || nm2.tabs[nm2.activeTab].id != 0 {
		t.Errorf("after closing last, activeTab = %d id = %d, want 0/0", nm2.activeTab, nm2.tabs[nm2.activeTab].id)
	}
}

func TestTabIDRouting(t *testing.T) {
	newIssues := []jira.Issue{{Key: "X-1"}}

	// Active tab A (id 10), inactive tab B (id 20). No projects so no follow-ups.
	base := func() model {
		m := newTabModel([]Tab{
			{id: 10, baseView: listView, board: boardState{jql: "a"}},
			{id: 20, baseView: listView, board: boardState{jql: "b"}},
		}, 0)
		m.projects = nil
		return m
	}

	// Result for inactive tab B -> snapshot only, flat unchanged.
	m := base()
	next, _ := m.Update(issuesLoadedMsg{issues: newIssues, tabID: 20})
	nm := next.(model)
	if len(nm.issues) != 0 {
		t.Errorf("flat issues should be unchanged for inactive-tab result, got %d", len(nm.issues))
	}
	if len(nm.tabs[1].board.issues) != 1 {
		t.Errorf("inactive tab snapshot not updated: %d", len(nm.tabs[1].board.issues))
	}

	// Result for active tab A -> flat updated.
	m = base()
	next, _ = m.Update(issuesLoadedMsg{issues: newIssues, tabID: 10})
	nm = next.(model)
	if len(nm.issues) != 1 {
		t.Errorf("flat issues should update for active-tab result, got %d", len(nm.issues))
	}

	// Result for unknown/closed tab -> dropped, no panic, no mutation.
	m = base()
	next, _ = m.Update(issuesLoadedMsg{issues: newIssues, tabID: 999})
	nm = next.(model)
	if len(nm.issues) != 0 || len(nm.tabs[0].board.issues) != 0 || len(nm.tabs[1].board.issues) != 0 {
		t.Errorf("result for unknown tab should be dropped")
	}
}

func TestDeriveEpicBoard(t *testing.T) {
	tests := []struct {
		name      string
		issue     *jira.Issue
		wantTitle string
		wantJQL   string
		wantErr   bool
	}{
		{"epic uses own key", &jira.Issue{Key: "EPIC-1", Type: "Epic"}, "EPIC-1", "parent = EPIC-1 ORDER BY status DESC", false},
		{"child uses parent key", &jira.Issue{Key: "DEV-2", Type: "Task", Parent: &jira.Parent{Key: "EPIC-9"}}, "EPIC-9", "parent = EPIC-9 ORDER BY status DESC", false},
		{"no parent errors", &jira.Issue{Key: "DEV-3", Type: "Task"}, "", "", true},
		{"nil errors", nil, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			title, jql, err := deriveEpicBoard(tt.issue)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if title != tt.wantTitle || jql != tt.wantJQL {
				t.Errorf("got (%q, %q), want (%q, %q)", title, jql, tt.wantTitle, tt.wantJQL)
			}
		})
	}
}

func TestBoardDedupe(t *testing.T) {
	jql := "parent = EPIC-1 ORDER BY status DESC"
	m := newTabModel([]Tab{
		{id: 0, baseView: listView, board: boardState{jql: "a"}},
		{id: 1, title: "EPIC-1", baseView: listView, board: boardState{jql: jql}},
	}, 0)

	next, _ := m.openBoardTab("EPIC-1", jql, tabEpicBoard)
	nm := next.(model)
	if len(nm.tabs) != 2 {
		t.Errorf("dedupe should not add a tab, got %d tabs", len(nm.tabs))
	}
	if nm.activeTab != 1 {
		t.Errorf("dedupe should focus the existing tab, activeTab = %d", nm.activeTab)
	}
}

func TestLayoutHeightReservation(t *testing.T) {
	m := model{windowWidth: 120, windowHeight: 40}

	list := m.calculateListLayout()
	wantList := 40 - 5 - 1 - tabBarHeight - ui.PanelOverheadHeight - listHeaderHeight
	if list.listHeight != wantList {
		t.Errorf("listHeight = %d, want %d (tab bar + column header reserved)", list.listHeight, wantList)
	}

	detail := m.calculateDetailLayout()
	// leftColumnFreeHeight = H - (metadata 7 + statusBar 1 + tabBar + overhead*2)
	wantDesc := (40 - (7 + 1 + tabBarHeight + ui.PanelOverheadHeight*2)) / 2
	if detail.descHeight != wantDesc {
		t.Errorf("descHeight = %d, want %d (tab bar reserved)", detail.descHeight, wantDesc)
	}
}

func TestKeyInterceptionGuards(t *testing.T) {
	twoTabs := func(mode viewMode, filtering bool) model {
		m := newTabModel([]Tab{
			{id: 0, baseView: listView, board: boardState{jql: "a"}},
			{id: 1, baseView: listView, board: boardState{jql: "b"}},
		}, 0)
		m.mode = mode
		m.filtering = filtering
		return m
	}

	// gt is a two-key sequence (g then t).
	gt := func(m model) model {
		next, _ := m.Update(keyPress("g"))
		next, _ = next.(model).Update(keyPress("t"))
		return next.(model)
	}

	// Base view, not filtering: `gt` switches.
	if got := gt(twoTabs(listView, false)).activeTab; got != 1 {
		t.Errorf("gt in base view should switch tab, activeTab = %d", got)
	}

	// Filtering: `gt` is input, not a switch.
	if got := gt(twoTabs(listView, true)).activeTab; got != 0 {
		t.Errorf("gt while filtering should not switch tab, activeTab = %d", got)
	}

	// Modal mode: `gt` ignored by tab handling.
	if got := gt(twoTabs(transitionView, false)).activeTab; got != 0 {
		t.Errorf("gt in a modal should not switch tab, activeTab = %d", got)
	}
}

func TestRenderTabBarNoPanic(t *testing.T) {
	for _, n := range []int{1, 3, 12} {
		tabs := make([]Tab, n)
		for i := range tabs {
			tabs[i] = Tab{id: i, title: "TAB-" + string(rune('A'+i))}
		}
		assertNoPanic(t, "renderTabBar", func() {
			m := newTabModel(tabs, 0)
			m.windowWidth = 40 // narrow
			if m.renderTabBar() == "" {
				t.Errorf("renderTabBar returned empty for %d tabs", n)
			}
		})
	}
}
