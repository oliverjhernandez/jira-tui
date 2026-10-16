package main

import (
	"slices"

	tea "charm.land/bubbletea/v2"
)

const navHistoryLimit = 50

type navEntry struct {
	mode     viewMode
	issueKey string
}

type navHistory struct {
	entries []navEntry
	pos     int
}

func (h *navHistory) visit(e navEntry) {
	if len(h.entries) == 0 {
		h.entries = []navEntry{{mode: listView}}
		h.pos = 0
	}
	if h.entries[h.pos] == e {
		return
	}
	h.entries = append(slices.Clip(h.entries[:h.pos+1]), e)
	if over := len(h.entries) - navHistoryLimit; over > 0 {
		h.entries = h.entries[over:]
	}
	h.pos = len(h.entries) - 1
}

func (h *navHistory) back() (navEntry, bool) {
	if h.pos <= 0 || h.pos >= len(h.entries) {
		return navEntry{}, false
	}
	h.pos--
	return h.entries[h.pos], true
}

func (h *navHistory) forward() (navEntry, bool) {
	if h.pos+1 >= len(h.entries) {
		return navEntry{}, false
	}
	h.pos++
	return h.entries[h.pos], true
}

func (m *model) visitIssue(key string) {
	m.history.visit(navEntry{mode: detailView, issueKey: key})
}

func (m model) historyBack() (tea.Model, tea.Cmd) {
	e, ok := m.history.back()
	if !ok {
		m.setInfo("No previous view")
		return m, m.clearStatusAfter(clearMsgTimeout)
	}
	return m.restoreNavEntry(e)
}

func (m model) historyForward() (tea.Model, tea.Cmd) {
	e, ok := m.history.forward()
	if !ok {
		m.setInfo("No next view")
		return m, m.clearStatusAfter(clearMsgTimeout)
	}
	return m.restoreNavEntry(e)
}

func (m model) restoreNavEntry(e navEntry) (tea.Model, tea.Cmd) {
	if e.mode == listView {
		m.detailReturnView = listView
		return m, m.leaveDetail()
	}

	m.activeIssue = nil
	m.focusedSection = metadataSection
	m.commentsCursor = 0
	m.worklogsCursor = 0
	m.subTasksCursor = 0
	m.detailLayout = m.calculateDetailLayout()
	m.mode = detailView
	m.loadingCount++
	return m, m.fetchIssueDetailCmd(e.issueKey)
}

func (m *model) leaveDetail() tea.Cmd {
	m.mode = m.detailReturnView
	m.detailReturnView = listView
	m.detailPolling = false
	m.activeIssue = nil

	m.descViewport.SetContent("")
	m.commentsViewport.SetContent("")
	m.worklogsViewport.SetContent("")
	m.issueLinksViewport.SetContent("")
	m.subTasksViewport.SetContent("")

	m.commentsCursor = 0
	m.worklogsCursor = 0
	m.subTasksCursor = 0
	m.loadingCount++
	return m.fetchMyIssuesCmd()
}
