package main

import (
	"testing"

	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func prioritiesFixture() []jira.Priority {
	return []jira.Priority{{ID: "1", Name: "High"}, {ID: "2", Name: "Medium"}, {ID: "3", Name: "Low"}}
}

func TestPriorityFromListStaysOnTheList(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "", issue("DEV-1", "alpha"), issue("DEV-2", "bravo"))
	m.priorities = prioritiesFixture()
	m.selectIssueByKey("DEV-2")

	next, _ := m.Update(keyPress("p"))
	pm := next.(model)

	if pm.mode != priorityView {
		t.Fatalf("p should open the priority modal, mode = %v", pm.mode)
	}
	if pm.previousMode != listView {
		t.Errorf("previousMode = %v, want listView so the modal returns to the board", pm.previousMode)
	}
	if pm.baseView != listView {
		t.Errorf("baseView = %v, want listView so the modal composites over the board", pm.baseView)
	}
	if pm.pendingIssue == nil || pm.pendingIssue.Key != "DEV-2" {
		t.Errorf("pendingIssue = %v, want the selected DEV-2", pm.pendingIssue)
	}

	// Completing the form must come back to the list, not jump to the detail view.
	pm.priorityData.SelectedPriority = "Low"
	pm.priorityData.Form.State = huh.StateCompleted
	done, _ := pm.updateEditPriorityView(nil)
	dm := done.(model)

	if dm.mode != listView {
		t.Errorf("after choosing a priority from the board, mode = %v, want listView", dm.mode)
	}
}

func TestPriorityFromDetailTargetsTheOpenIssue(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "", issue("DEV-1", "alpha"), issue("DEV-2", "bravo"))
	m.priorities = prioritiesFixture()

	// The board selection is DEV-2, but the user drilled into DEV-1.
	m.selectIssueByKey("DEV-2")
	m.setPendingIssue(m.selectedIssue)
	m.mode = detailView
	m.baseView = detailView
	m.activeIssue = &jira.Issue{Key: "DEV-1", Summary: "alpha", Priority: jira.Priority{Name: "High"}}

	next, _ := m.Update(keyPress("p"))
	pm := next.(model)

	if pm.mode != priorityView {
		t.Fatalf("p should open the priority modal, mode = %v", pm.mode)
	}
	if pm.pendingIssue == nil {
		t.Fatal("no pending issue")
	}
	if pm.pendingIssue.Key != "DEV-1" {
		t.Errorf("priority would be applied to %s, but the detail view is showing DEV-1",
			pm.pendingIssue.Key)
	}
}

func TestPriorityFromDetailReturnsToDetail(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "", issue("DEV-1", "alpha"))
	m.priorities = prioritiesFixture()
	m.mode = detailView
	m.baseView = detailView
	m.activeIssue = &jira.Issue{Key: "DEV-1", Priority: jira.Priority{Name: "High"}}

	next, _ := m.Update(keyPress("p"))
	pm := next.(model)

	pm.priorityData.SelectedPriority = "Low"
	pm.priorityData.Form.State = huh.StateCompleted
	done, _ := pm.updateEditPriorityView(nil)
	dm := done.(model)

	if dm.mode != detailView {
		t.Errorf("mode = %v, want detailView", dm.mode)
	}
}

func TestPriorityLoadingCountIsBalanced(t *testing.T) {
	t.Parallel()

	// Opening and cancelling must not leak the counter: a leaked count silently
	// disables ctrl+r, which guards on loadingCount > 0.
	m := filteringModel(t, "", issue("DEV-1", "alpha"))
	m.priorities = prioritiesFixture()
	m.selectIssueByKey("DEV-1")

	start := m.loadingCount

	opened, _ := m.Update(keyPress("p"))
	om := opened.(model)
	if om.loadingCount != start {
		t.Errorf("opening the modal moved loadingCount %d -> %d", start, om.loadingCount)
	}

	cancelled, _ := om.updateEditPriorityView(keyPress("esc"))
	cm := cancelled.(model)
	if cm.loadingCount != start {
		t.Errorf("cancelling leaked loadingCount: %d -> %d", start, cm.loadingCount)
	}
	if cm.mode != listView {
		t.Errorf("esc returned to %v, want listView", cm.mode)
	}
}

func TestPriorityPostThenResultNetsToZero(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "", issue("DEV-1", "alpha"))
	m.priorities = prioritiesFixture()
	m.selectIssueByKey("DEV-1")
	start := m.loadingCount

	opened, _ := m.Update(keyPress("p"))
	om := opened.(model)
	om.priorityData.SelectedPriority = "Low"
	om.priorityData.Form.State = huh.StateCompleted

	posted, _ := om.updateEditPriorityView(nil)
	pm := posted.(model)
	if pm.loadingCount != start+1 {
		t.Errorf("posting should raise loadingCount by exactly 1, got %d -> %d", start, pm.loadingCount)
	}

	// priorityPostedMsg consumes the post's count and afterIssueAction adds one
	// back for the board refresh it queues, so exactly one load stays in flight.
	settled, _ := pm.Update(priorityPostedMsg{})
	sm := settled.(model)
	if sm.loadingCount != start+1 {
		t.Errorf("loadingCount = %d after the post settled, want %d (the queued refresh)",
			sm.loadingCount, start+1)
	}

	// That refresh settling must bring it back to baseline, not below it.
	done, _ := sm.Update(issuesLoadedMsg{issues: nil, tabID: 0})
	dm := done.(model)
	if dm.loadingCount < 0 {
		t.Errorf("loadingCount went negative: %d", dm.loadingCount)
	}
}
