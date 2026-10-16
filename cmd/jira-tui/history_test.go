package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func issueEntry(key string) navEntry {
	return navEntry{mode: detailView, issueKey: key}
}

var listEntry = navEntry{mode: listView}

func TestNavHistory(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		run     func(h *navHistory) (navEntry, bool)
		want    navEntry
		wantOK  bool
		entries []navEntry
	}{
		{
			name:   "back on empty history",
			run:    func(h *navHistory) (navEntry, bool) { return h.back() },
			wantOK: false,
		},
		{
			name: "first visit seeds the list as root",
			run: func(h *navHistory) (navEntry, bool) {
				h.visit(issueEntry("A"))
				return h.back()
			},
			want:    listEntry,
			wantOK:  true,
			entries: []navEntry{listEntry, issueEntry("A")},
		},
		{
			name: "back then forward",
			run: func(h *navHistory) (navEntry, bool) {
				h.visit(issueEntry("A"))
				h.visit(issueEntry("B"))
				h.back()
				return h.forward()
			},
			want:    issueEntry("B"),
			wantOK:  true,
			entries: []navEntry{listEntry, issueEntry("A"), issueEntry("B")},
		},
		{
			name: "forward at the end",
			run: func(h *navHistory) (navEntry, bool) {
				h.visit(issueEntry("A"))
				return h.forward()
			},
			wantOK:  false,
			entries: []navEntry{listEntry, issueEntry("A")},
		},
		{
			name: "repeat visit is ignored",
			run: func(h *navHistory) (navEntry, bool) {
				h.visit(issueEntry("A"))
				h.visit(issueEntry("A"))
				return h.back()
			},
			want:    listEntry,
			wantOK:  true,
			entries: []navEntry{listEntry, issueEntry("A")},
		},
		{
			name: "visit after back drops the forward entries",
			run: func(h *navHistory) (navEntry, bool) {
				h.visit(issueEntry("A"))
				h.visit(issueEntry("B"))
				h.back()
				h.visit(issueEntry("C"))
				return h.forward()
			},
			wantOK:  false,
			entries: []navEntry{listEntry, issueEntry("A"), issueEntry("C")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var h navHistory
			got, ok := tt.run(&h)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("got (%+v, %v), want (%+v, %v)", got, ok, tt.want, tt.wantOK)
			}
			if len(h.entries) != len(tt.entries) {
				t.Fatalf("entries = %+v, want %+v", h.entries, tt.entries)
			}
			for i := range tt.entries {
				if h.entries[i] != tt.entries[i] {
					t.Errorf("entries[%d] = %+v, want %+v", i, h.entries[i], tt.entries[i])
				}
			}
		})
	}
}

func TestNavHistoryLimit(t *testing.T) {
	t.Parallel()

	var h navHistory
	for i := range navHistoryLimit + 10 {
		h.visit(issueEntry(string(rune('a' + i%26))))
	}
	if len(h.entries) != navHistoryLimit {
		t.Errorf("len(entries) = %d, want %d", len(h.entries), navHistoryLimit)
	}
	if h.pos != len(h.entries)-1 {
		t.Errorf("pos = %d, want %d", h.pos, len(h.entries)-1)
	}
}

func TestHistoryKeysBetweenListAndDetail(t *testing.T) {
	t.Parallel()

	m := newTabModel([]Tab{
		{id: 0, title: "Board", baseView: listView, board: boardState{jql: "a"}},
	}, 0)
	m.issues = []jira.Issue{{Key: "DEV-1", Status: "In Progress", Project: jira.Project{ID: "P"}}}
	m.sections = []Section{{Name: "All", Issues: m.issues}}

	next, _ := m.updateListView(keyPress("enter"))
	m = next.(model)
	if m.mode != detailView {
		t.Fatalf("after enter mode = %v, want detailView", m.mode)
	}

	next, _ = m.updateDetailView(keyPress("H"))
	m = next.(model)
	if m.mode != listView {
		t.Fatalf("after H mode = %v, want listView", m.mode)
	}

	next, cmd := m.updateListView(keyPress("L"))
	m = next.(model)
	if m.mode != detailView {
		t.Fatalf("after L mode = %v, want detailView", m.mode)
	}
	if cmd == nil {
		t.Error("L should fetch the issue it returns to")
	}

	next, _ = m.updateDetailView(keyPress("L"))
	m = next.(model)
	if m.mode != detailView {
		t.Errorf("L at the end of history changed mode to %v", m.mode)
	}
}

func TestHistoryIsPerTab(t *testing.T) {
	t.Parallel()

	m := newTabModel([]Tab{
		{id: 0, title: "A", baseView: listView, board: boardState{jql: "a"}},
		{id: 1, title: "B", baseView: listView, board: boardState{jql: "b"}},
	}, 0)
	m.visitIssue("DEV-1")
	m.saveActiveTab()

	m.activeTab = 1
	m.loadActiveTab()
	if _, ok := m.history.back(); ok {
		t.Error("tab B should start with no history")
	}
	m.saveActiveTab()

	m.activeTab = 0
	m.loadActiveTab()
	if e, ok := m.history.back(); !ok || e != listEntry {
		t.Errorf("tab A back = (%+v, %v), want list", e, ok)
	}
}
