package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateListView(msg tea.Msg) (tea.Model, tea.Cmd) {

	if keyMsg, ok := msg.(tea.KeyMsg); ok {

		issuesToShow := m.issues
		if m.filterInput.Value() != "" {
			issuesToShow = filterIssues(m.issues, m.filterInput.Value())
		}

		if m.filtering {
			switch keyMsg.String() {
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

		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < len(issuesToShow) {
					return m, nil
				}
			}
		case "down", "j":
			if m.cursor < len(issuesToShow)-1 {
				m.cursor++
				if m.cursor < len(issuesToShow) {
					return m, nil
				}
			}
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

				width := m.windowWidth - 10
				height := m.windowHeight - 15
				vp := viewport.New(width, height)
				m.detailViewport = &vp
				return m, m.fetchIssueDetail(m.selectedIssue.Key)
			}
		case "esc":
			m.filterInput.SetValue("")
			m.cursor = 0
		}

	}

	return m, nil
}

func (m model) renderListView() string {
	log.Printf("=== renderListView called ===")

	panelWidth := max(120, m.windowWidth-4)
	panelHeight := m.windowHeight - 4

	listPanelStyle := ui.BaseListPanelStyle.
		Height(panelHeight).
		Width(panelWidth)

	var b strings.Builder
	b.WriteString("My Jira Issues\n\n")

	issuesToShow := m.issues
	if m.filterInput.Value() != "" {
		issuesToShow = filterIssues(m.issues, m.filterInput.Value())
	}

	maxVisible := panelHeight - 4
	start := 0
	end := min(len(issuesToShow), maxVisible)

	if m.cursor >= end {
		start = m.cursor - maxVisible + 1
		end = m.cursor + 1
	} else if m.cursor < start {
		start = m.cursor
		end = start + maxVisible
	}

	var listContent strings.Builder
	for i := start; i < end; i++ {
		issue := issuesToShow[i]

		key := ui.KeyFieldStyle.Render(fmt.Sprintf("[%s]", issue.Key))
		summary := ui.SummaryFieldStyle.Render(truncateLongString(issue.Summary, 40))
		statusBadge := ui.StatusFieldStyle.Render(renderStatusBadge(issue.Status))
		assignee := ui.AssigneeFieldStyle.Render(issue.Assignee)
		priority := ui.PriorityFieldStyle.Render(issue.Priority)

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
	statusBarRender := ui.StatusBarStyle.Render(statusBar)

	return listRender + "\n" + statusBarRender
}
