package main

import (
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
	var cmd tea.Cmd

	var detailViewSections = []focusedSection{
		descSection,
		commentsSection,
		worklogsSection,
		epicChildrenSection,
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
				m.commentsViewport.ScrollDown(1)
				return m, nil
			case "k":
				m.commentsViewport.ScrollUp(1)
				return m, nil
			}
		}

		switch keyPressMsg.String() {
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

			currentIdx := findIndex(m.focusedSection, detailViewSections)
			m.focusedSection = detailViewSections[(currentIdx+1)%len(detailViewSections)]
			return m, nil

		case "shift+tab":
			currentIdx := findIndex(m.focusedSection, detailViewSections)
			m.focusedSection = detailViewSections[(currentIdx-1+len(detailViewSections))%len(detailViewSections)]
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
		}
	}

	return m, cmd
}

func (m model) renderDetailView() string {
	if m.issueDetail == nil {
		return ui.PanelStyleActive.Render("Loading issue...")
	}

	panelWidth := ui.GetAvailableWidth(m.windowWidth)

	infoPanel := m.renderInfoPanel(panelWidth)
	metadataPanel := m.renderMetadataPanel(m.detailLayout.leftColumnWidth)
	descriptionPanel := m.renderDescriptionPanel(m.detailLayout.leftColumnWidth, m.detailLayout.descHeight)
	commentsPanel := m.renderCommentsPanel(m.detailLayout.leftColumnWidth, m.detailLayout.commentsHeight)
	statusBar := m.renderDetailStatusBar()

	leftColumn := lipgloss.JoinVertical(lipgloss.Left, metadataPanel, descriptionPanel, commentsPanel)

	return infoPanel + "\n" + leftColumn + "\n" + statusBar
}
