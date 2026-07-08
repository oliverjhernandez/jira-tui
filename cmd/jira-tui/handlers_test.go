package main

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func keyPress(s string) tea.KeyPressMsg {
	switch s {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		return tea.KeyPressMsg{Code: tea.KeyTab}
	default:
		return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
	}
}

func assertNoPanic(t *testing.T, name string, fn func()) {
	t.Helper()
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("%s panicked: %v", name, r)
		}
	}()
	fn()
}

func TestUpdateListView_NavigationOnEmptySections(t *testing.T) {
	// sections is nil (state before issues have loaded).
	for _, k := range []string{"j", "k", "up", "down", "G"} {
		assertNoPanic(t, "updateListView "+k, func() {
			m := model{}
			m.updateListView(keyPress(k))
		})
	}
}

func TestUpdateListView_YankOnEmptySections(t *testing.T) {
	for _, k := range []string{"k", "K", "s"} {
		assertNoPanic(t, "updateListView yank "+k, func() {
			m := model{lastKey: "y"}
			m.updateListView(keyPress(k))
		})
	}
}

func TestUpdateDetailView_NilActiveIssue(t *testing.T) {
	keys := []string{"p", "l", "w", "t", "e", "a", "enter", "G", "k", "d", "tab"}
	for _, k := range keys {
		assertNoPanic(t, "updateDetailView(nil activeIssue) "+k, func() {
			m := model{mode: detailView}
			m.updateDetailView(keyPress(k))
		})
	}
}

func TestUpdateDetailView_EmptyCollections(t *testing.T) {
	// activeIssue present but no comments/worklogs/subtasks.
	base := func(section focusedSection) model {
		return model{
			mode:           detailView,
			focusedSection: section,
			activeIssue:    &jira.Issue{Key: "DEV-1", Project: jira.Project{Key: "DEV"}},
		}
	}

	cases := []struct {
		name    string
		section focusedSection
		keys    []string
	}{
		{"comments", commentsSection, []string{"d", "e", "j", "k"}},
		{"worklogs", worklogsSection, []string{"d", "e", "j", "k"}},
		{"subtasks", subTasksSection, []string{"t", "a", "enter", "j", "k"}},
	}

	for _, tc := range cases {
		for _, k := range tc.keys {
			assertNoPanic(t, tc.name+" "+k, func() {
				m := base(tc.section)
				m.updateDetailView(keyPress(k))
			})
		}
	}
}

func TestUpdateDetailView_CommentYankEmpty(t *testing.T) {
	assertNoPanic(t, "detail comment yy empty", func() {
		m := model{
			mode:           detailView,
			focusedSection: commentsSection,
			lastKey:        "y",
			activeIssue:    &jira.Issue{Key: "DEV-1"},
		}
		m.updateDetailView(keyPress("y"))
	})
}

func TestUpdateTransitionView_EscReturnsToPreviousMode(t *testing.T) {
	m := model{
		mode:         transitionView,
		previousMode: detailView,
	}
	next, _ := m.updateTransitionView(keyPress("esc"))
	nm := next.(model)
	if nm.mode != detailView {
		t.Errorf("esc should return to previousMode detailView, got %v", nm.mode)
	}
}

func TestUpdateTransitionView_NilDataNoPanic(t *testing.T) {
	assertNoPanic(t, "updateTransitionView nil data", func() {
		m := model{mode: transitionView, transitionData: nil}
		m.updateTransitionView(keyPress("j"))
	})
}
