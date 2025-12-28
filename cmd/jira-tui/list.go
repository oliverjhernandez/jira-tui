package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
)

func (m model) updateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {

	issuesToShow := m.issues
	if m.filterInput.Value() != "" {
		issuesToShow = filterIssues(m.issues, m.filterInput.Value())
	}

	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filterInput.SetValue("")
			m.filterInput.Blur()
			m.cursor = 0
			return m, nil
		case "enter":
			m.filtering = false
			m.filterInput.Blur()
			return m, nil
		}

		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)

		return m, cmd
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < len(issuesToShow) {
				return m, m.fetchIssueDetail(issuesToShow[m.cursor].Key)
			}
		}
	case "down", "j":
		if m.cursor < len(issuesToShow)-1 {
			m.cursor++
			if m.cursor < len(issuesToShow) {
				return m, m.fetchIssueDetail(issuesToShow[m.cursor].Key)
			}
		}
	case "esc":
		m.filterInput.SetValue("")
		m.cursor = 0
	case "/":
		m.filtering = true
		m.filterInput.SetValue("")
		m.filterInput.Focus()
		m.cursor = 0
		return m, textinput.Blink
	case "enter":
		if len(issuesToShow) > 0 && m.cursor < len(issuesToShow) {
			m.selectedIssue = &issuesToShow[m.cursor]
			m.mode = detailView
			m.loadingDetail = true
			m.issueDetail = nil
			return m, m.fetchIssueDetail(m.selectedIssue.Key)
		}
	}

	return m, nil
}

func (m model) renderListView() string {

	panelWidth := max(120, m.windowWidth-4)
	panelHeight := m.windowHeight - 4

	listPanelStyle := baseListPanelStyle.
		Height(panelHeight).
		Width(panelWidth)

	var b strings.Builder
	b.WriteString("My Jira Issues\n\n")

	issuesToShow := m.issues
	if m.filterInput.Value() != "" {
		issuesToShow = filterIssues(m.issues, m.filterInput.Value())
	}

	var listContent strings.Builder
	for i, issue := range issuesToShow {
		key := keyFieldStyle.Render(fmt.Sprintf("[%s]", issue.Key))
		summary := summaryFieldStyle.Render(truncate(issue.Summary, 40))
		statusBadge := statusFieldStyle.Render(renderStatusBadge(issue.Status))
		assignee := assigneeFieldStyle.Render(issue.Assignee)
		priority := priorityFieldStyle.Render(issue.Priority)

		line := key + " " + summary + " " + statusBadge + " " + assignee + " " + priority

		if m.cursor == i {
			line = "> " + line
		} else {
			line = " " + line
		}

		listContent.WriteString(line + "\n")
	}

	var statusBar string
	if m.filtering {
		statusBar = "Filter: " + m.filterInput.View() + " (enter to finish, esc to cancel)"
	} else if m.filterInput.Value() != "" {
		statusBar = fmt.Sprintf("Filtered by: '%s' (%d/%d) | / to change | esc to clear", m.filterInput.Value(), len(issuesToShow), len(m.issues))
	} else {
		statusBar = "\n/ filter | enter detail | t transition | q quit"
	}

	listRender := listPanelStyle.Render(listContent.String())
	statusBarRender := statusBarStyle.Render(statusBar)

	return listRender + "\n" + statusBarRender
}
