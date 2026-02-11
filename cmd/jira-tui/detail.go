package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type rightColumnView int

const (
	worklogsView rightColumnView = iota
	epicChildrenView
)

func (m model) updateDetailView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	var detailViewSections = []focusedSection{
		descriptionSection,
		commentsSection,
		worklogsSection,
		epicChildrenSection,
	}

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
			// if m.issueDetail.Type == "Epic" && len(m.epicChildren) > 0 {
			// 	m.toggleRightColumnView()
			// } FIX: do something to show epic children

			m.focusedSectionIndex = (m.focusedSectionIndex + 1) % len(detailViewSections)
			m.focusedSection = detailViewSections[m.focusedSectionIndex]
			return m, nil

		case "shift+tab":
			m.focusedSectionIndex--
			if m.focusedSectionIndex < 0 {
				m.focusedSectionIndex = len(detailViewSections) - 1
			}
			m.focusedSection = detailViewSections[m.focusedSectionIndex]

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

		case "l":
			m.mode = issueSearchView
			m.searchData = NewSearchFormData()
			m.issueSelectionMode = linkIssue
			return m, m.searchData.Form.Init()

		case "L":
			m.mode = detailView
			m.issueSelectionMode = linkIssue
			if m.issueDetail.IsLinkedToChange {
				m.loadingDetail = true
				return m, m.unlinkIssue(m.issueDetail.ChangeIssueLinkID)
			}
			return m, nil

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
	leftColumnWidth := int(float64(panelWidth)*0.6) - 6

	index := ui.StatusBarDescStyle.Render(
		fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.sections[m.sectionCursor].Issues)),
	)

	parent := ""
	if m.issueDetail.Parent != nil {
		parent = ui.RenderIssueType(m.issueDetail.Parent.Type, false) + " " +
			ui.StatusBarDescStyle.Render(m.issueDetail.Parent.Key+" / ")
	}

	issueKey := ui.RenderIssueType(m.issueDetail.Type, false) + " " + ui.DetailHeaderStyle.Render(m.issueDetail.Key)
	summaryMaxWidth := 50
	issueSummary := ui.DetailValueStyle.Render(truncateLongString(m.issueDetail.Summary, summaryMaxWidth))
	var linkedIssue string
	if m.issueDetail.IsLinkedToChange {
		linkedIssue = "🔗 " + jira.MonthlyChangeIssue
	}

	leftHeaderLine1 := index + " " + parent + issueKey + "  " + issueSummary + " " + linkedIssue

	status := ui.RenderStatusBadge(m.issueDetail.Status)
	assignee := ui.StatusBarDescStyle.Render("@" + strings.ToLower(strings.Split(m.issueDetail.Assignee, " ")[0]))

	logged := ""
	if m.selectedIssueWorklogs != nil {
		logged = ui.StatusBarDescStyle.Render("Logged: " + extractLoggedTime(m.selectedIssueWorklogs))
	}

	leftHeaderLine2 := status + "  " + assignee + "  " + logged
	leftHeader := leftHeaderLine1 + "\n" + leftHeaderLine2

	colwidth := 30

	col1 := ui.RenderFieldStyled("Priority", ui.RenderPriority(m.issueDetail.Priority.Name, true), colwidth)
	col2 := ui.RenderFieldStyled("Reporter", m.issueDetail.Reporter, colwidth)
	col3 := ui.RenderFieldStyled("Type", ui.RenderIssueType(m.issueDetail.Type, true), colwidth)
	metadataRow1 := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	col4 := ui.RenderFieldStyled("Created", timeAgo(m.issueDetail.Created), colwidth)
	col5 := ui.RenderFieldStyled("Updated", timeAgo(m.issueDetail.Updated), colwidth)
	metadataRow2 := lipgloss.JoinHorizontal(lipgloss.Top, col4, col5)

	metadataRow := metadataRow1 + "\n" + metadataRow2

	var main strings.Builder
	main.WriteString(leftHeader + "\n\n")
	main.WriteString(metadataRow + "\n\n")

	detailPanel := ui.PanelStyleActive.
		Width(leftColumnWidth).
		Render(main.String())

	infoPanelHeight := lipgloss.Height(infoPanel)
	detailPanelHeight := lipgloss.Height(detailPanel)
	statusBarHeight := 1

	panelsHeight := infoPanelHeight + detailPanelHeight + statusBarHeight + 6 // newlines between sections

	m.detailViewport.Width = leftColumnWidth
	m.detailViewport.Height = m.windowHeight - panelsHeight

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

	var commentsSectionStyle lipgloss.Style
	if m.focusedSection == commentsSection {
		commentsSectionStyle = ui.PanelActiveStyle
	} else {
		commentsSectionStyle = ui.PanelInactiveStyle
	}

	m.detailViewport.SetContent(scrollContent.String())
	leftViewport := m.detailViewport.View()

	styledLeftViewport := commentsSectionStyle.
		Width(leftColumnWidth).
		Render(leftViewport)

	var statusBar strings.Builder
	if m.statusMessage != "" {
		statusBar.WriteString(m.statusMessage)
	} else if m.loadingDetail || m.loadingTransitions {
		statusBar.WriteString(m.spinner.View() + "Loading...")
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

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, detailPanel, styledLeftViewport)

	return infoPanel + "\n" + leftColumn + "\n\n" + ui.StatusBarStyle.Render(statusBar.String())
}
