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

type rightColumnView int

const (
	worklogsView rightColumnView = iota
	epicChildrenView
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
			m.mode = descriptionView
			m.editingDescription = true
			return m, m.descriptionData.Form.Init()
		case "p":
			m.priorityData = NewPriorityFormData(m.priorityOptions, m.issueDetail.Priority.Name)
			m.mode = priorityView
			m.editingPriority = true
			return m, m.priorityData.Form.Init()
		case "tab":
			if m.issueDetail.Type == "Epic" && len(m.epicChildren) > 0 {
				m.toggleRightColumnView()
			}
			return m, nil
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
			m.textArea = textarea.New()
			m.textArea.Placeholder = "Add a comment..."
			m.textArea.Focus()
			m.textArea.SetWidth(80)
			m.commentData = NewCommentFormData()
			m.mode = commentView
			return m, m.commentData.Form.Init()
		case "w":
			m.worklogData = NewWorklogFormData()
			m.mode = worklogView
			return m, m.worklogData.Form.Init()
		case "a":
			m.mode = userSearchView
			m.loadingAssignUsers = true
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.cursor = 0
			return m, m.fetchUsers(m.issueDetail.Key)
		case "e":
			m.mode = estimateView
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
			m.textArea.SetValue("")
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

	infoPanel := m.renderInfoPanel()

	panelWidth := ui.GetPanelWidth(m.windowWidth)
	leftColumnWidth := int(float64(panelWidth) * 0.6)
	rightColumnWidth := int(float64(panelWidth) * 0.4)

	m.detailViewport.Width = leftColumnWidth

	panelsHeight := 6 + // infoPanel height
		11 + // detailPanel height
		1 // statusBar height
	m.detailViewport.Height = m.windowHeight - panelsHeight
	leftViewport := m.detailViewport.View()

	index := ui.StatusBarDescStyle.Render(fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.sections[m.sectionCursor].Issues)))

	parent := ""
	if m.issueDetail.Parent != nil {
		parent = ui.RenderIssueType(m.issueDetail.Parent.Type, false) + " " +
			ui.StatusBarDescStyle.Render(m.issueDetail.Parent.Key+" / ")
	}

	issueKey := ui.RenderIssueType(m.issueDetail.Type, false) + " " + ui.DetailHeaderStyle.Render(m.issueDetail.Key)
	summaryMaxWidth := 50
	issueSummary := ui.DetailValueStyle.Render(truncateLongString(m.issueDetail.Summary, summaryMaxWidth))

	leftHeaderLine1 := index + " " + parent + issueKey + "  " + issueSummary

	status := ui.RenderStatusBadge(m.issueDetail.Status)
	assignee := ui.StatusBarDescStyle.Render("@" + strings.ToLower(strings.Split(m.issueDetail.Assignee, " ")[0]))

	logged := ""
	if m.selectedIssueWorklogs != nil {
		logged = ui.StatusBarDescStyle.Render("Logged: " + extractLoggedTime(m.selectedIssueWorklogs))
	}

	leftHeaderLine2 := status + "  " + assignee + "  " + logged

	leftHeader := leftHeaderLine1 + "\n" + leftHeaderLine2

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
		ui.SeparatorStyle.Render(strings.Repeat("─", 20)) + "\n\n")

	if m.issueDetail.Description != "" {
		wrappedDesc := ui.DetailValueStyle.Width(leftColumnWidth - 4).Render(m.issueDetail.Description)
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
			wrappedBody := ui.CommentBodyStyle.Width(leftColumnWidth - 4).Render(c.Body)
			scrollContent.WriteString(wrappedBody + "\n")

			if i < commentCount-1 {
				scrollContent.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
			} else {
				scrollContent.WriteString("\n")
			}
		}
	}

	styledScrollContent := ui.PanelSecondaryStyle.
		Width(leftColumnWidth).
		Render(scrollContent.String())

	m.detailViewport.SetContent(styledScrollContent)

	var worklogs strings.Builder

	for i, w := range m.selectedIssueWorklogs {
		logHours := w.Time / 60 / 60
		timeAgo := timeAgo(w.UpdatedAt)

		worklogs.WriteString(ui.CommentTimestampStyle.Render(strconv.Itoa(logHours)+"h"+" • "+timeAgo) + "\n")
		worklogs.WriteString(w.Author.AccountID + "\n") // TODO: map to proper user name
		worklogs.WriteString("\"" + w.Description + "\"" + "\n")
		if i != len(m.selectedIssueWorklogs)-1 {
			worklogs.WriteString(strings.Repeat("-", 10) + "\n")
		}
	}

	var epicChildren strings.Builder

	if m.epicChildren != nil {
		for _, ec := range m.epicChildren {
			epicChildren.WriteString(ui.RenderIssueType(ec.Type, false) + " ")
			epicChildren.WriteString(ui.DetailHeaderStyle.Render(ec.Key) + " ")
			epicChildren.WriteString(ui.StatusBarDescStyle.Render("@"+strings.ToLower(strings.Split(ec.Assignee, " ")[0])) + "\n")
			epicChildren.WriteString(truncateLongString(ec.Summary, summaryMaxWidth) + "\n")

		}
	}

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
	main.WriteString(leftHeader + "\n\n")
	main.WriteString(metadataRow + "\n\n")

	if m.loadingDetail || m.loadingTransitions {
		statusBar.Reset()
		statusBar.WriteString(m.spinner.View() + "Loading...")
	}

	detailPanel := ui.PanelStyleActive.
		Width(leftColumnWidth).
		Render(main.String())

	worklogsPanel := ui.PanelStyleActive.Width(rightColumnWidth).Render(worklogs.String())
	epicChildrenPanel := ui.PanelStyleActive.Width(rightColumnWidth).Render(epicChildren.String())

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, detailPanel, leftViewport)

	var rightColumn string
	switch m.rightColumnView {
	case worklogsView:
		rightColumn = worklogsPanel
	case epicChildrenView:
		rightColumn = epicChildrenPanel
	}

	both := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	return infoPanel + "\n" + both + "\n\n" + ui.StatusBarStyle.Render(statusBar.String())
}
