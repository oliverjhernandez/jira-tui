package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateDetailView(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf(">>> updateDetailView called with: %T", msg)
	var cmd tea.Cmd

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
		case "c":
			m.mode = postCommentView
			m.postingComment = true
			m.editTextArea.SetValue("")
			m.editTextArea.Focus()
			return m, textarea.Blink
		case "a":
			m.mode = assignableUsersSearchView
			m.loadingAssignableUsers = true
			m.filterInput.SetValue("")
			m.filterInput.Focus()
			m.cursor = 0
			return m, m.fetchAssignableUsers(m.selectedIssue.Key)
		case "esc":
			m.mode = listView
			m.selectedIssue = nil
			m.issueDetail = nil
			m.loading = true
			return m, m.fetchMyIssues()
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
		return ui.PanelStyleActive.Render("Loading issue...")
	}

	panelWidth := max(120, m.windowWidth-4)
	panelHeight := m.windowHeight - 4
	contentWidth := panelWidth - 6 // padding and border

	selectedIssue := m.issueDetail

	index := ui.StatusBarDescStyle.Render(fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.issues)))

	parent := ""
	if selectedIssue.Parent != nil {
		parent = ui.StatusBarDescStyle.Render(selectedIssue.Parent.ID + " / ")
	}

	issueKey := ui.DetailHeaderStyle.Render(selectedIssue.Key)
	issueSummary := ui.DetailValueStyle.Render(truncateLongString(selectedIssue.Summary, 50))

	headerLine1 := index + " " + parent + issueKey + "  " + issueSummary

	status := ui.RenderStatusBadge(selectedIssue.Status)
	assignee := ui.StatusBarDescStyle.Render("@" + strings.Split(selectedIssue.Assignee, " ")[0])
	estimate := ui.StatusBarDescStyle.Render("Est: " + selectedIssue.OriginalEstimate)

	logged := ""
	if m.selectedIssueWorklogs != nil {
		logged = ui.StatusBarDescStyle.Render("Logged: " + extractLoggedTime(m.selectedIssueWorklogs))
	}

	headerLine2 := status + "  " + assignee + "  " + estimate + "  " + logged

	header := headerLine1 + "\n" + headerLine2

	col1 := ui.RenderFieldStyled("Priority", selectedIssue.Priority.Name, contentWidth/3)
	col2 := ui.RenderFieldStyled("Reporter", m.issueDetail.Reporter, contentWidth/3)
	col3 := ui.RenderFieldStyled("Type", selectedIssue.Type, contentWidth/3)
	metadataRow := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	var scrollContent strings.Builder

	scrollContent.WriteString(ui.SectionTitleStyle.Render("─── Description ") +
		ui.SeparatorStyle.Render(strings.Repeat("─", contentWidth-16)) + "\n\n")

	if m.issueDetail.Description != "" {
		scrollContent.WriteString(ui.DetailValueStyle.Render(m.issueDetail.Description) + "\n\n")
	} else {
		scrollContent.WriteString(ui.StatusBarDescStyle.Render("No description") + "\n\n")
	}

	commentCount := len(m.issueDetail.Comments)
	scrollContent.WriteString(ui.SectionTitleStyle.Render(fmt.Sprintf("─── Comments (%d) ", commentCount)) +
		ui.SeparatorStyle.Render(strings.Repeat("─", contentWidth-18)) + "\n\n")

	if commentCount > 0 {
		for _, c := range m.issueDetail.Comments {
			author := ui.CommentAuthorStyle.Render(c.Author)
			timestamp := ui.CommentTimestampStyle.Render(" • " + timeAgo(c.Created))
			scrollContent.WriteString(author + timestamp + "\n")
			scrollContent.WriteString(ui.CommentBodyStyle.Render(c.Body) + "\n\n")
		}
	} else {
		scrollContent.WriteString(ui.StatusBarDescStyle.Render("No comments yet") + "\n")
	}

	m.detailViewport.SetContent(scrollContent.String())

	statusBar := strings.Join([]string{
		ui.RenderKeyBind("a", "assignee"),
		ui.RenderKeyBind("c", "comment"),
		ui.RenderKeyBind("e", "edit"),
		ui.RenderKeyBind("t", "transition"),
		ui.RenderKeyBind("esc", "back"),
	}, "  ")

	var output strings.Builder
	output.WriteString(header + "\n\n")
	output.WriteString(metadataRow + "\n\n")
	output.WriteString(m.detailViewport.View())

	detailPanel := ui.PanelStyleActive.
		Width(panelWidth).
		Height(panelHeight).
		Render(output.String())

	return detailPanel + "\n" + ui.StatusBarStyle.Render(statusBar)
}
