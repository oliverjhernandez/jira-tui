package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = listView
		m.selectedIssue = nil
		m.issueDetail = nil
		m.loading = true
		return m, m.fetchData
	case "d":
		m.mode = editDescriptionView
		m.editingDescription = true
		m.editTextArea.SetValue(m.issueDetail.Description)
		m.editTextArea.Focus()
		return m, textarea.Blink
	case "p":
		m.mode = editPriorityView
		m.editingPriority = true
		priorityIndex := -1
		for i, p := range m.priorityOptions {
			if p.Name == m.issueDetail.Priority.Name {
				priorityIndex = i
				break
			}
		}
		m.priorityCursor = max(0, priorityIndex)
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
	log.Printf("=== renderDetailView called ===")

	if m.selectedIssue == nil || m.issueDetail == nil {
		return "Loading issue...\n"
	}

	var detailContent strings.Builder
	selectedIssue := m.issueDetail

	index := "[" + strconv.Itoa(m.cursor) + "/" + strconv.Itoa(len(m.issues)) + "]"
	parent := "NA"
	if selectedIssue.Parent != nil {
		parent = selectedIssue.Parent.ID
	}

	issueKey := selectedIssue.Key
	issueSummary := truncate(selectedIssue.Summary, 40)
	status := renderStatusBadge(selectedIssue.Status)
	assignee := strings.Split(selectedIssue.Assignee, " ")[0]
	estimate := selectedIssue.OriginalEstimate
	logged := "4h" // TODO: pending

	header := index + " " + parent + "/" + issueKey + " " + issueSummary + "\n" + " " + assignee + " " + estimate + " " + logged

	detailContent.WriteString(header + "\n")

	detailContent.WriteString(ui.SeparatorStyle.Render("") + "\n")
	col1 := (renderField("Status", status))
	col2 := renderField("Assignee", m.issueDetail.Assignee)
	col3 := renderField("Created", "XXXXXXX") + "\n"

	row1 := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)
	detailContent.WriteString(row1 + "\n")

	col1 = renderField("Priority", selectedIssue.Priority.Name)
	col2 = renderField("Reporter", m.issueDetail.Reporter)
	col3 = renderField("Updated", "XXXXXXX")

	row2 := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)
	detailContent.WriteString(row2 + "\n")

	detailContent.WriteString(renderField("Type", selectedIssue.Type) + "\n")
	detailContent.WriteString(ui.SeparatorStyle.Render("") + "\n")

	detailContent.WriteString(ui.DetailLabelStyle.Render("Description:") + "\n")
	desc := m.issueDetail.Description
	if len(desc) > 200 {
		desc = desc[:200] + "..."
	}
	detailContent.WriteString(ui.DetailValueStyle.Render(desc) + "\n\n")

	if len(m.issueDetail.Comments) > 0 {
		detailContent.WriteString(ui.DetailLabelStyle.Render(fmt.Sprintf("Comments: (%d):", len(m.issueDetail.Comments))) + "\n")
		detailContent.WriteString(ui.DetailValueStyle.Render("Press Enter for full view") + "\n")
	}

	var statusBar string
	if m.filtering {
		statusBar = "Filter: " + m.filterInput.View() + " (enter to finish, esc to cancel)"
	} else {
		statusBar = "\n/ filter | enter detail | t transition | q quit"
	}

	detailRender := ui.DetailPanelStyle.Render(detailContent.String())
	statusBarRender := ui.StatusBarStyle.Render(statusBar)

	return detailRender + "\n\n" + statusBarRender
}
