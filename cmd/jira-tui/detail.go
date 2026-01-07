package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateDetailView(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf(">>> updateDetailView called with: %T", msg)
	var cmd tea.Cmd

	// beforeOffset := m.detailViewport.YOffset
	// m.detailViewport, cmd = m.detailViewport.Update(msg)
	// afterOffset := m.detailViewport.YOffset

	// if beforeOffset != afterOffset {
	// 	log.Printf("YOffset changed! %d -> %d", beforeOffset, afterOffset)
	// }

	if keyPressMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyPressMsg.String() {
		case "j":
			m.detailViewport.ScrollDown(1)
		case "k":
			m.detailViewport.ScrollUp(1)
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
		case "esc":
			m.mode = listView
			m.selectedIssue = nil
			m.issueDetail = nil
			m.loading = true
			return m, m.fetchData
		case "q", "ctrl+c":
			return m, tea.Quit
		default:
			log.Printf("Default case - passing key '%s' to viewport", keyPressMsg.String())
			vp, cmd := m.detailViewport.Update(msg)
			m.detailViewport = &vp
			log.Printf("After viewport.Update - YOffset: %d", m.detailViewport.YOffset)
			return m, cmd
		}
	}

	return m, cmd
}

func (m model) renderDetailView() string {
	if m.selectedIssue == nil || m.issueDetail == nil {
		return "Loading issue...\n"
	}

	// HEADER
	selectedIssue := m.issueDetail

	index := "[" + strconv.Itoa(m.cursor+1) + "/" + strconv.Itoa(len(m.issues)) + "]"
	parent := "NA"
	if selectedIssue.Parent != nil {
		parent = selectedIssue.Parent.ID
	}

	issueKey := selectedIssue.Key
	issueSummary := truncateLongString(selectedIssue.Summary, 40)
	status := renderStatusBadge(selectedIssue.Status)
	assignee := strings.Split(selectedIssue.Assignee, " ")[0]
	estimate := selectedIssue.OriginalEstimate
	logged := "4h" // TODO: get from tempo api

	header := index + " " + parent + "/" + issueKey + " " + issueSummary + "\n" + " " + assignee + " " + estimate + " " + logged

	// 	METADATA
	col1 := (renderField("Status", status))
	col2 := renderField("Assignee", m.issueDetail.Assignee)
	col3 := renderField("Created", "XXXXXXX") // TODO: get from api
	row1 := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	col1 = renderField("Priority", selectedIssue.Priority.Name)
	col2 = renderField("Reporter", m.issueDetail.Reporter)
	col3 = renderField("Updated", "XXXXXXX") // TODO: get from api
	row2 := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	metadata := row1 + "\n" + row2 + "\n" + renderField("Type", selectedIssue.Type)

	// TODO: place type somewhere
	// metadataContent.WriteString(renderField("Type", selectedIssue.Type) + "\n")
	// metadataContent.WriteString(ui.SeparatorStyle.Render("") + "\n")

	var scrollContent strings.Builder
	scrollContent.WriteString("--- Description -------------------------\n")
	scrollContent.WriteString(ui.DetailValueStyle.Render(m.issueDetail.Description) + "\n\n")

	scrollContent.WriteString("--- Comments -------------------------\n")
	if len(m.issueDetail.Comments) > 0 {
		for _, c := range m.issueDetail.Comments {
			scrollContent.WriteString(fmt.Sprintf("\n%s • %s\n", ui.CommentAuthorStyle.Render(c.Author), ui.CommentTimestampStyle.Render(timeAgo(c.Created))))
			scrollContent.WriteString(c.Body + "\n")
		}
	}

	var statusBar string
	if m.filtering {
		statusBar = "Filter: " + m.filterInput.View() + " (enter to finish, esc to cancel)"
	} else {
		statusBar = "\n/ filter | enter detail | t transition | q quit"
	}

	m.detailViewport.SetContent(scrollContent.String())
	log.Printf("Viewport - Width: %d, Height: %d, Total lines: %d, YOffset: %d",
		m.detailViewport.Width,
		m.detailViewport.Height,
		m.detailViewport.TotalLineCount(),
		m.detailViewport.YOffset)

	var output strings.Builder
	output.WriteString(header + "\n")
	output.WriteString(ui.SeparatorStyle.Render("") + "\n")
	output.WriteString(metadata + "\n")
	output.WriteString(ui.SeparatorStyle.Render("") + "\n")
	output.WriteString(m.detailViewport.View() + "\n")
	output.WriteString(ui.StatusBarStyle.Render(statusBar))

	panelWidth := max(120, m.windowWidth-4)
	panelHeight := m.windowHeight - 4

	detailPanelStyle := ui.BaseDetailPanelStyle.
		Height(panelHeight).
		Width(panelWidth)

	return detailPanelStyle.Render(output.String())
}
