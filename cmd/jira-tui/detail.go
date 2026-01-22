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
			if m.issueDetail != nil {
				m.mode = transitionView
				m.loadingTransitions = true
				m.transitionCursor = 0
				return m, m.fetchTransitions(m.issueDetail.Key)
			}
		case "c":
			m.mode = postCommentView
			m.postingComment = true
			m.editTextArea.Focus()
			return m, nil
		case "w":
			m.worklogData = NewWorklogFormData()
			m.mode = postWorklogView
			m.postingComment = true
			return m, m.worklogData.Form.Init()
		case "a":
			m.mode = assignableUsersSearchView
			m.loadingAssignableUsers = true
			m.filterInput.SetValue("")
			m.filterInput.Focus()
			m.cursor = 0
			return m, m.fetchAssignableUsers(m.issueDetail.Key)
		case "esc":
			m.mode = listView
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
	if m.issueDetail == nil {
		return ui.PanelStyleActive.Render("Loading issue...")
	}

	infoPanelHeight := 5 // 2 content lines + 2 border lines + 1 newline
	panelWidth := max(120, m.windowWidth-4)
	panelHeight := m.windowHeight - 2 - infoPanelHeight
	contentWidth := panelWidth - 6 // padding and border

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

	commandsHelp := strings.Join([]string{
		ui.RenderKeyBind("a", "assignee"),
		ui.RenderKeyBind("c", "comment"),
		ui.RenderKeyBind("e", "edit"),
		ui.RenderKeyBind("t", "transition"),
		ui.RenderKeyBind("esc", "back"),
	}, "  ")

	var statusBar strings.Builder
	statusBar.WriteString(header + "\n\n")
	statusBar.WriteString(metadataRow + "\n\n")
	statusBar.WriteString(m.detailViewport.View())

	if m.loadingTransitions {
		statusBar.WriteString(m.spinner.View() + "Loading transitions...\n")
	}

	detailPanel := ui.PanelStyleActive.
		Width(panelWidth).
		Height(panelHeight).
		Render(statusBar.String())

	infoPanel := m.renderInfoPanel()
	return infoPanel + "\n" + detailPanel + "\n" + ui.StatusBarStyle.Render(commandsHelp)
}
