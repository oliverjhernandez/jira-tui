package main

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateDetailView(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf(">>> updateDetailView called with: %T", msg)
	var cmd tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyMsg); ok {
		m.statusMessage = ""

		switch keyPressMsg.String() {
		case "j":
			m.detailViewport.ScrollDown(1)
		case "k":
			m.detailViewport.ScrollUp(1)
		case "d":
			m.descriptionData = NewDescriptionFormData(m.issueDetail.Description)
			m.mode = editDescriptionView
			m.editingDescription = true
			return m, m.descriptionData.Form.Init()
		case "p":
			m.priorityData = NewPriorityFormData(m.priorityOptions, m.issueDetail.Priority.Name)
			m.mode = editPriorityView
			m.editingPriority = true
			return m, m.priorityData.Form.Init()
		case "t":
			if m.issueDetail != nil {
				if m.issueDetail.Description == "" {
					m.statusMessage = "Cannot transition, missing description."
					return m, nil
				}

				if m.issueDetail.OriginalEstimate == "" {
					m.statusMessage = "Cannot transition, missing original estimate"
					return m, nil
				}

				m.mode = transitionView
				m.loadingTransitions = true
				m.transitionCursor = 0
				return m, m.fetchTransitions(m.issueDetail.Key)
			}
		case "c":
			m.commentData = NewCommentFormData()
			m.mode = postCommentView
			return m, m.commentData.Form.Init()
		case "w":
			m.worklogData = NewWorklogFormData()
			m.mode = postWorklogView
			return m, m.worklogData.Form.Init()
		case "a":
			m.mode = assignUsersSearchView
			m.loadingAssignUsers = true
			m.statusBarInput.SetValue("")
			m.statusBarInput.Focus()
			m.cursor = 0
			return m, m.fetchAssignUsers(m.issueDetail.Key)
		case "e":
			m.mode = postEstimateView
			m.estimateData = NewEstimateFormData()
			return m, m.estimateData.Form.Init()
		case "ctrl+r":
			if m.loadingDetail {
				return m, nil
			}
			m.loadingDetail = true
			return m, m.fetchIssueDetail(m.issueDetail.Key)
		case "esc":
			m.mode = listView
			m.issueDetail = nil
			m.editTextArea.SetValue("")
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
	if m.issueDetail == nil {
		return ui.PanelStyleActive.Render("Loading issue...")
	}

	panelWidth := ui.GetPanelWidth(m.windowWidth)
	panelHeight := ui.GetPanelHeight(m.windowHeight)
	contentWidth := panelWidth - 6

	index := ui.StatusBarDescStyle.Render(fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.sections[m.sectionCursor].Issues)))

	parent := ""
	if m.issueDetail.Parent != nil {
		parent = ui.RenderIssueType(m.issueDetail.Parent.Type, false) + " " +
			ui.StatusBarDescStyle.Render(m.issueDetail.Parent.Key+" / ")
	}

	issueKey := ui.RenderIssueType(m.issueDetail.Type, false) + " " + ui.DetailHeaderStyle.Render(m.issueDetail.Key)
	summaryMaxWidth := contentWidth - 30
	issueSummary := ui.DetailValueStyle.Render(truncateLongString(m.issueDetail.Summary, summaryMaxWidth))

	headerLine1 := index + " " + parent + issueKey + "  " + issueSummary

	status := ui.RenderStatusBadge(m.issueDetail.Status)
	assignee := ui.StatusBarDescStyle.Render("@" + strings.ToLower(strings.Split(m.issueDetail.Assignee, " ")[0]))

	logged := ""
	if m.selectedIssueWorklogs != nil {
		logged = ui.StatusBarDescStyle.Render("Logged: " + extractLoggedTime(m.selectedIssueWorklogs))
	}

	headerLine2 := status + "  " + assignee + "  " + logged

	header := headerLine1 + "\n" + headerLine2

	col1 := ui.RenderFieldStyled("Priority", ui.RenderPriority(m.issueDetail.Priority.Name, true), 30)
	col2 := ui.RenderFieldStyled("Reporter", m.issueDetail.Reporter, 30)
	col3 := ui.RenderFieldStyled("Type", ui.RenderIssueType(m.issueDetail.Type, true), 30)
	metadataRow1 := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	col4 := ui.RenderFieldStyled("Created", timeAgo(m.issueDetail.Created), 30)
	col5 := ui.RenderFieldStyled("Updated", timeAgo(m.issueDetail.Updated), 30)
	metadataRow2 := lipgloss.JoinHorizontal(lipgloss.Top, col4, col5)

	metadataRow := metadataRow1 + "\n" + metadataRow2

	var scrollContent strings.Builder

	scrollContent.WriteString(ui.SeparatorStyle.Render(strings.Repeat("─", 4)+" ") +
		ui.SectionTitleStyle.Render("󰠮 Description ") +
		ui.SeparatorStyle.Render(strings.Repeat("─", 60)) + "\n\n")

	if m.issueDetail.Description != "" {
		wrappedDesc := ui.DetailValueStyle.Width(contentWidth - 4).Render(m.issueDetail.Description)
		scrollContent.WriteString(wrappedDesc + "\n\n")
	} else {
		scrollContent.WriteString(ui.StatusBarDescStyle.Render("No description") + "\n\n")
	}

	commentCount := len(m.issueDetail.Comments)
	scrollContent.WriteString(ui.SeparatorStyle.Render(strings.Repeat("─", 4)+" ") +
		ui.SectionTitleStyle.Render(fmt.Sprintf("󱅰 Comments (%d) ", commentCount)) +
		ui.SeparatorStyle.Render(strings.Repeat("─", 60)) + "\n\n")

	if commentCount > 0 {
		for i, c := range m.issueDetail.Comments {
			author := ui.CommentAuthorStyle.Render(c.Author)
			timestamp := ui.CommentTimestampStyle.Render(" • " + timeAgo(c.Created))
			scrollContent.WriteString(author + timestamp + "\n")
			wrappedBody := ui.CommentBodyStyle.Width(contentWidth - 4).Render(c.Body)
			scrollContent.WriteString(wrappedBody + "\n")

			if i < commentCount-1 {
				scrollContent.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
			} else {
				scrollContent.WriteString("\n")
			}
		}
	}

	m.detailViewport.SetContent(scrollContent.String())

	var statusBar strings.Builder

	if m.statusMessage != "" {
		statusBar.WriteString(m.statusMessage)
	} else {
		statusBar.WriteString(strings.Join([]string{
			ui.RenderKeyBind("j/k", "scroll"),
			ui.RenderKeyBind("d", "description"),
			ui.RenderKeyBind("p", "priority"),
			ui.RenderKeyBind("c", "comment"),
			ui.RenderKeyBind("w", "worklog"),
			ui.RenderKeyBind("a", "assignee"),
			ui.RenderKeyBind("t", "transition"),
			ui.RenderKeyBind("esc", "back"),
			ui.RenderKeyBind("q", "quit"),
		}, "  "))
	}

	var main strings.Builder
	main.WriteString(header + "\n\n")
	main.WriteString(metadataRow + "\n\n")
	main.WriteString(m.detailViewport.View())

	if m.loadingDetail || m.loadingTransitions {
		statusBar.Reset()
		statusBar.WriteString(m.spinner.View() + "Loading...")
	}

	detailPanel := ui.PanelStyleActive.
		Width(panelWidth).
		Height(panelHeight).
		Render(main.String())

	infoPanel := m.renderInfoPanel()
	return infoPanel + "\n" + detailPanel + "\n" + ui.StatusBarStyle.Render(statusBar.String())
}
