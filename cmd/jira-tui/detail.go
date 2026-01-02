package main

import (
	"encoding/json"
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

	if m.selectedIssue == nil {
		log.Printf("DEBUG: Returning early - selectedIssue nil? %v",
			m.selectedIssue == nil)
		return "Loading issue...\n"
	} else {
		log.Printf("Selected: %+v", m.selectedIssue)
	}

	if m.issueDetail == nil {
		log.Printf("DEBUG: Returning early - issueDetail nil? %v",
			m.issueDetail == nil)
		return "Loading issue...\n"
	} else {
		detailJSON, _ := json.MarshalIndent(m.issueDetail, "", "  ")
		log.Printf("Detail JSON: %s", string(detailJSON))
	}

	var detailContent strings.Builder
	selectedIssue := m.issueDetail

	index := "[" + strconv.Itoa(m.cursor) + "/" + strconv.Itoa(len(m.issues)) + "]"
	parent := "No parent"
	if selectedIssue.Parent != nil {
		parent = selectedIssue.Parent.ID
	}

	status := renderStatusBadge(selectedIssue.Status)
	assignee := strings.Split(selectedIssue.Assignee, " ")[0]
	estimate := selectedIssue.OriginalEstimate
	logged := "4h" // TODO: pending

	header := index + " " + parent + " " + status + " " + assignee + " " + estimate + " " + logged

	detailContent.WriteString(header + "\n\n")
	detailContent.WriteString(renderField("Summary", truncate(selectedIssue.Summary, 40)) + "\n")
	detailContent.WriteString(renderField("Type", selectedIssue.Type) + "\n")
	detailContent.WriteString(renderField("Priority", selectedIssue.Priority.Name) + "\n")

	if m.loadingDetail {
		detailContent.WriteString("Loading details...\n")
	} else if m.issueDetail != nil {

		if m.issueDetail != nil && m.issueDetail.Key == selectedIssue.Key {
			detailContent.WriteString(renderField("Assignee", m.issueDetail.Assignee) + "\n")
			detailContent.WriteString(renderField("Reporter", m.issueDetail.Reporter) + "\n")

			if m.issueDetail.Description != "" {
				detailContent.WriteString(ui.DetailLabelStyle.Render("Description:") + "\n")
				desc := m.issueDetail.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				detailContent.WriteString(ui.DetailValueStyle.Render(desc) + "\n\n")
			}

			if len(m.issueDetail.Comments) > 0 {
				detailContent.WriteString(ui.DetailLabelStyle.Render(fmt.Sprintf("Comments: (%d):", len(m.issueDetail.Comments))) + "\n")
				detailContent.WriteString(ui.DetailValueStyle.Render("Press Enter for full view") + "\n")
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

	detailRender := ui.DetailPanelStyle.Render(detailContent.String())
	statusBarRender := ui.StatusBarStyle.Render(statusBar)

	return detailRender + "\n\n" + statusBarRender
}
