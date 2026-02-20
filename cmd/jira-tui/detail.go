package main

import (
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateDetailView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	var detailViewSections = []focusedSection{
		descSection,
		commentsSection,
		worklogsSection,
		childrenSection,
	}

	if keyPressMsg, ok := msg.(tea.KeyMsg); ok {
		m.statusMessage = ""

		switch m.focusedSection {
		case descSection:
			switch keyPressMsg.String() {
			case "j":
				m.descViewport.ScrollDown(1)
				return m, nil
			case "k":
				m.descViewport.ScrollUp(1)
				return m, nil
			}
		case commentsSection:
			switch keyPressMsg.String() {

			case "j":
				if m.commentsCursor < len(m.issueDetail.Comments)-1 {
					m.commentsCursor++
				}

				cursorLine := m.getCommentCursorLine()
				m.commentsViewport.SetYOffset(cursorLine)

				commentsContent := m.buildCommentsContent(m.detailLayout.leftColumnWidth)
				m.commentsViewport.SetContent(commentsContent)

				return m, nil

			case "k":
				if m.commentsCursor > 0 {
					m.commentsCursor--
				}

				cursorLine := m.getCommentCursorLine()
				m.commentsViewport.SetYOffset(cursorLine)

				commentsContent := m.buildCommentsContent(m.detailLayout.leftColumnWidth)
				m.commentsViewport.SetContent(commentsContent)
				return m, nil

			case "c":
				m.textArea = textarea.New()
				m.textArea.Placeholder = "Add a comment..."
				m.textArea.Focus()
				m.textArea.SetWidth(100)
				m.mode = commentView
				return m, nil

			case "e":
				m.textArea = textarea.New()
				textAreaWidth := 100
				m.textArea.SetWidth(textAreaWidth)
				comment := jira.ExtractText(m.issueDetail.Comments[m.commentsCursor].Body, textAreaWidth)
				m.textArea.SetValue(comment)
				m.textArea.Focus()
				m.editingComment = true
				m.mode = commentView
				return m, nil

			case "d":
				cmd := m.deleteComment(m.issueDetail.Key, m.issueDetail.Comments[m.commentsCursor].ID)
				return m, cmd
			}

		case worklogsSection:
			switch keyPressMsg.String() {
			case "j":
				m.worklogsViewport.ScrollDown(1)
				return m, nil
			case "k":
				m.worklogsViewport.ScrollUp(1)
				return m, nil
			}

		case childrenSection:
			switch keyPressMsg.String() {
			case "j":
				m.childrenViewport.ScrollDown(1)
				return m, nil
			case "k":
				m.childrenViewport.ScrollUp(1)
				return m, nil
			}

		}

		switch keyPressMsg.String() {
		case "d":
			descText := jira.ExtractText(m.issueDetail.Description, m.detailLayout.leftColumnWidth)
			m.descriptionData = NewDescriptionFormData(descText)
			m.mode = descriptionView
			m.editingDescription = true
			return m, m.descriptionData.Form.Init()

		case "p":
			m.priorityData = NewPriorityFormData(m.priorityOptions, m.issueDetail.Priority.Name)
			m.mode = priorityView
			m.editingPriority = true
			m.loadingDetail = true
			return m, m.priorityData.Form.Init()

		case "tab":
			currentIdx := findIndex(m.focusedSection, detailViewSections)
			m.focusedSection = detailViewSections[(currentIdx+1)%len(detailViewSections)]
			return m, nil

		case "shift+tab":
			currentIdx := findIndex(m.focusedSection, detailViewSections)
			m.focusedSection = detailViewSections[(currentIdx-1+len(detailViewSections))%len(detailViewSections)]
			return m, nil

		case "t":
			if m.issueDetail != nil {
				if m.issueDetail.Description == nil {
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
				m.loadingDetail = true
				return m, m.fetchTransitions(m.issueDetail.Key)
			}

		case "l":
			m.mode = issueSearchView
			m.searchData = NewSearchFormData()
			m.issueSelectionMode = linkIssue
			m.loadingDetail = true
			return m, m.searchData.Form.Init()

		case "L":
			m.mode = detailView
			m.issueSelectionMode = linkIssue
			if m.issueDetail.IsLinkedToChange {
				m.loadingDetail = true
				return m, m.unlinkIssue(m.issueDetail.ChangeIssueLinkID)
			}
			return m, nil

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
			m.commentsCursor = 0
			return m, m.fetchMyIssues()
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, cmd
}

func (m model) renderDetailView() string {
	if m.issueDetail == nil {
		return ui.PanelStyleActive.Render("Loading issue...")
	}

	metadataPanel := m.renderMetadataPanel(m.detailLayout.leftColumnWidth)
	descriptionPanel := m.renderDescriptionPanel(m.detailLayout.leftColumnWidth)
	commentsPanel := m.renderCommentsPanel(m.detailLayout.leftColumnWidth)

	worklogPanel := m.renderWorklogsPanel(m.detailLayout.rightColumnWidth)
	childrenPanel := m.renderChildrenPanel(m.detailLayout.rightColumnWidth)

	statusBar := m.renderDetailStatusBar()

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, metadataPanel, descriptionPanel, commentsPanel)
	rightColumn := lipgloss.JoinVertical(lipgloss.Right, worklogPanel, childrenPanel)

	columns := lipgloss.JoinHorizontal(lipgloss.Top, leftColumn, rightColumn)

	return columns + "\n" + statusBar
}
