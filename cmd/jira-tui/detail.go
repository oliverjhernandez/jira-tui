package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m model) updateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = listView
		m.selectedIssue = nil
		m.issueDetail = nil
	case "e":
		m.mode = editDescriptionView
		m.editingDescription = true
		m.editTextArea.SetValue(m.issueDetail.Description)
		m.editTextArea.Focus()
		return m, textarea.Blink
	case "t":
		if m.selectedIssue != nil {
			m.mode = transitionView
			m.loadingTransitions = true
			m.transitionCursor = 0
			return m, m.fetchTransitions(m.selectedIssue.Key)
		}
	}

	return m, nil
}

func (m model) renderDetailView() string {
	if m.selectedIssue == nil {
		return "No issue selected\n"
	}

	var detailContent strings.Builder
	selectedIssue := m.issues[m.cursor]

	header := detailHeaderStyle.Render(selectedIssue.Key) + " " + renderStatusBadge(selectedIssue.Status)
	detailContent.WriteString(header + "\n\n")
	detailContent.WriteString(renderField("Summary", truncate(selectedIssue.Summary, 40)) + "\n")
	detailContent.WriteString(renderField("Type", selectedIssue.Type) + "\n")

	if m.loadingDetail {
		detailContent.WriteString("Loading details...\n")
	} else if m.issueDetail != nil {

		if m.issueDetail != nil && m.issueDetail.Key == selectedIssue.Key {
			detailContent.WriteString(renderField("Assignee", m.issueDetail.Assignee) + "\n")
			detailContent.WriteString(renderField("Reporter", m.issueDetail.Reporter) + "\n")

			if m.issueDetail.Description != "" {
				detailContent.WriteString(detailLabelStyle.Render("Description:") + "\n")
				desc := m.issueDetail.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				detailContent.WriteString(detailValueStyle.Render(desc) + "\n\n")
			}

			if len(m.issueDetail.Comments) > 0 {
				detailContent.WriteString(detailLabelStyle.Render(fmt.Sprintf("Comments: (%d):", len(m.issueDetail.Comments))) + "\n")
				detailContent.WriteString(detailValueStyle.Render("Press Enter for full view") + "\n")
			}
		} else {
			detailContent.WriteString("\n" + lipgloss.NewStyle().Faint(true).Render("Press Enter for full details") + "\n")
		}
	}

	var statusBar string
	if m.filtering {
		statusBar = "Filter: " + m.filterInput.View() + " (enter to finish, esc to cancel)"
	} else {
		statusBar = "\n/ filter | enter detail | t transition | q quit"
	}

	detailRender := detailPanelStyle.Render(detailContent.String())
	statusBarRender := statusBarStyle.Render(statusBar)

	return detailRender + "\n\n" + statusBarRender
}
