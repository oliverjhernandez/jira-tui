package main

import (
	"strings"
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestViewModeIsModal(t *testing.T) {
	baseViews := []viewMode{listView, detailView}
	modalViews := []viewMode{
		newIssueView, transitionView, userSearchView, descriptionView,
		priorityView, commentView, worklogView, issueLinkView, estimateView,
		cancelReasonView, blockReasonView, issueSearchView,
	}

	for _, v := range baseViews {
		if v.isModal() {
			t.Errorf("%v.isModal() = true, want false", v)
		}
	}
	for _, v := range modalViews {
		if !v.isModal() {
			t.Errorf("%v.isModal() = false, want true", v)
		}
	}
}

func TestUpdateTracksBaseView(t *testing.T) {
	// Loading an issue detail switches to the detailView base view; the wrapper
	// should record it as the current base.
	m := model{
		mode:         listView,
		baseView:     listView,
		windowWidth:  100,
		windowHeight: 40,
		activeTab:    0,
		tabs:         []Tab{{id: 0, title: "My Issues", baseView: listView}},
	}

	next, _ := m.Update(issueDetailLoadedMsg{detail: &jira.Issue{Key: "DEV-1"}, tabID: 0})
	nm := next.(model)

	if nm.mode != detailView {
		t.Fatalf("mode = %v, want detailView", nm.mode)
	}
	if nm.baseView != detailView {
		t.Errorf("baseView = %v, want detailView (should follow the base view)", nm.baseView)
	}
}

func TestUpdateBaseViewUnchangedForModal(t *testing.T) {
	// A message that leaves the app on a modal view must not overwrite baseView.
	m := model{
		mode:         transitionView,
		baseView:     detailView,
		windowWidth:  100,
		windowHeight: 40,
	}

	// keyTimeoutMsg is a no-op that keeps the current (modal) mode.
	next, _ := m.Update(keyTimeoutMsg{})
	nm := next.(model)

	if nm.baseView != detailView {
		t.Errorf("baseView = %v, want detailView (unchanged while on a modal)", nm.baseView)
	}
}

func TestRenderBackgroundSelectsBaseView(t *testing.T) {
	assertNoPanic(t, "renderBackground detail", func() {
		m := model{
			baseView:     detailView,
			windowWidth:  100,
			windowHeight: 40,
			activeIssue:  &jira.Issue{Key: "DEV-1"},
		}
		m.detailLayout = m.calculateDetailLayout()
		if out := m.renderBackground(); strings.TrimSpace(out) == "" {
			t.Errorf("renderBackground() returned empty for detail base view")
		}
	})

	assertNoPanic(t, "renderBackground list", func() {
		m := model{
			baseView:     listView,
			windowWidth:  100,
			windowHeight: 40,
		}
		m.listLayout = m.calculateListLayout()
		_ = m.renderBackground()
	})
}

func TestRenderModalNoPanic(t *testing.T) {
	assertNoPanic(t, "renderModal", func() {
		m := model{
			baseView:     listView,
			windowWidth:  100,
			windowHeight: 40,
		}
		m.listLayout = m.calculateListLayout()
		if out := m.renderModal("Test", "some content", 0.3, 0.3); out == "" {
			t.Errorf("renderModal returned empty output")
		}
	})
}
