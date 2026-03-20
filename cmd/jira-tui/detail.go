package main

import (
	"strconv"
	"time"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateDetailView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	var detailViewSections = []focusedSection{
		metadataSection,
		descriptionSection,
		commentsSection,
		worklogsSection,
		childrenSection,
	}

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		m.statusMessage = ""

		switch m.focusedSection {
		case metadataSection:
			switch {
			case keyPressMsg.String() == "y" && m.lastKey == "":
				m.lastKey = "y"
				tick := tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
					return keyTimeoutMsg{}
				})
				return m, tick

			case keyPressMsg.String() == "k" && m.lastKey == "y":
				m.lastKey = ""
				textToCopy := m.issueDetail.Key
				yankToClipboard(textToCopy)
				m.statusMessage = "Key yanked to clipboard"
				return m, nil

			case keyPressMsg.String() == "K" && m.lastKey == "y":
				m.lastKey = ""
				textToCopy := jiraURL + m.issueDetail.Key
				yankToClipboard(textToCopy)
				m.statusMessage = "URL yanked to clipboard"
				return m, nil

			case keyPressMsg.String() == "s" && m.lastKey == "y":
				m.lastKey = ""
				textToCopy := m.issueDetail.Summary
				yankToClipboard(textToCopy)
				m.statusMessage = "Summary yanked to clipboard"
				return m, nil

			case keyPressMsg.String() == "t":
				m.activeIssue = &jira.Issue{
					Key:              m.issueDetail.Key,
					Description:      m.issueDetail.Description,
					OriginalEstimate: m.issueDetail.OriginalEstimate,
					Type:             m.issueDetail.Type,
				}

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

			case keyPressMsg.String() == "a":
				m.activeIssue = &jira.Issue{
					Key: m.issueDetail.Key,
				}
				m.mode = userSearchView
				m.loadingAssignUsers = true
				m.textInput.SetValue("")
				m.textInput.Focus()
				m.cursor = 0
				return m, m.fetchAssignableUsers(m.issueDetail.Key)

			}

		case descriptionSection:
			switch {
			case keyPressMsg.String() == "j":
				m.descViewport.ScrollDown(1)
				return m, nil
			case keyPressMsg.String() == "k":
				m.descViewport.ScrollUp(1)
				return m, nil

			case keyPressMsg.String() == "y" && m.lastKey == "":
				m.lastKey = "y"
				tick := tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
					return keyTimeoutMsg{}
				})
				return m, tick

			case keyPressMsg.String() == "y" && m.lastKey == "y":
				m.lastKey = ""
				textToCopy := jira.ExtractText(m.issueDetail.Description, m.detailLayout.leftColumnWidth)
				yankToClipboard(textToCopy)
				m.statusMessage = "Description yanked to clipboard"

				return m, nil
			}
		case commentsSection:
			switch {

			case keyPressMsg.String() == "y" && m.lastKey == "":
				m.lastKey = "y"
				tick := tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
					return keyTimeoutMsg{}
				})
				return m, tick

			case keyPressMsg.String() == "y" && m.lastKey == "y":
				m.lastKey = ""
				textToCopy := jira.ExtractText(m.issueDetail.Comments[m.commentsCursor].Body, m.detailLayout.leftColumnWidth)
				yankToClipboard(textToCopy)
				m.statusMessage = "Comment yanked to clipboard"
				return m, nil

			case keyPressMsg.String() == "j":
				if m.commentsCursor < len(m.issueDetail.Comments)-1 {
					m.commentsCursor++
				}

				cursorLine := m.getCommentCursorLine()
				m.commentsViewport.SetYOffset(cursorLine)

				commentsContent := m.buildCommentsContent(m.detailLayout.leftColumnWidth)
				m.commentsViewport.SetContent(commentsContent)

				return m, nil

			case keyPressMsg.String() == "k":
				if m.commentsCursor > 0 {
					m.commentsCursor--
				}

				cursorLine := m.getCommentCursorLine()
				m.commentsViewport.SetYOffset(cursorLine)

				commentsContent := m.buildCommentsContent(m.detailLayout.leftColumnWidth)
				m.commentsViewport.SetContent(commentsContent)
				return m, nil

			case keyPressMsg.String() == "c":
				m.textArea = textarea.New()
				m.textArea.Placeholder = "Add a comment..."
				m.textArea.Focus()
				m.textArea.SetWidth(100)
				m.mode = commentView
				return m, nil

			case keyPressMsg.String() == "e":
				m.textArea = textarea.New()
				textAreaWidth := 100
				m.textArea.SetWidth(textAreaWidth)
				var comment string
				if m.issueDetail.Comments != nil {
					comment = jira.ExtractText(m.issueDetail.Comments[m.commentsCursor].Body, textAreaWidth)
				}
				m.textArea.SetValue(comment)
				m.textArea.Focus()
				m.editingComment = true
				m.mode = commentView
				return m, nil

			case keyPressMsg.String() == "d":
				cmd := m.deleteComment(m.issueDetail.Key, m.issueDetail.Comments[m.commentsCursor].ID)
				return m, cmd
			}

		case worklogsSection:
			switch {

			case keyPressMsg.String() == "j":
				if m.worklogsCursor < len(m.issueDetail.Worklogs)-1 {
					m.worklogsCursor++
				}

				cursorLine := m.worklogsCursor * 4
				m.worklogsViewport.SetYOffset(cursorLine)

				wlContent := m.buildWorklogsContent(m.detailLayout.rightColumnWidth)
				m.worklogsViewport.SetContent(wlContent)

				return m, nil

			case keyPressMsg.String() == "k":
				if m.worklogsCursor > 0 {
					m.worklogsCursor--
				}

				cursorLine := m.worklogsCursor * 4
				m.worklogsViewport.SetYOffset(cursorLine)

				wlContent := m.buildWorklogsContent(m.detailLayout.rightColumnWidth)
				m.worklogsViewport.SetContent(wlContent)
				return m, nil

			case keyPressMsg.String() == "e":
				m.editingWorklog = true
				m.worklogFormData = m.NewWorklogForm(&m.issueDetail.Worklogs[m.worklogsCursor], 40)
				m.mode = worklogView

				return m, m.worklogFormData.Form.Init()

			case keyPressMsg.String() == "d":
				cmd := m.deleteWorkLog(strconv.Itoa(m.issueDetail.Worklogs[m.worklogsCursor].ID))
				return m, cmd
			}

		case childrenSection:
			switch {

			case keyPressMsg.String() == "j":
				if m.childrenCursor < len(m.issueDetail.Children)-1 {
					m.childrenCursor++
				}

				cursorLine := m.childrenCursor * 6
				m.childrenViewport.SetYOffset(cursorLine)

				chContent := m.buildChildrenContent(m.detailLayout.rightColumnWidth)
				m.childrenViewport.SetContent(chContent)

				return m, nil

			case keyPressMsg.String() == "k":
				if m.childrenCursor > 0 {
					m.childrenCursor--
				}

				cursorLine := m.childrenCursor * 4
				m.childrenViewport.SetYOffset(cursorLine)

				chContent := m.buildChildrenContent(m.detailLayout.rightColumnWidth)
				m.childrenViewport.SetContent(chContent)
				return m, nil

			case keyPressMsg.String() == "c":
				i := &NewIssueFormData{
					ParentKey: m.issueDetail.Key,
				}
				m.newIssueData = m.NewIssueForm(i)
				m.mode = newIssueView
				return m, m.newIssueData.Form.Init()

			case keyPressMsg.String() == "t":
				m.activeIssue = &m.issueDetail.Children[m.childrenCursor]

				if m.issueDetail.Children[m.childrenCursor].Description == nil {
					m.statusMessage = "Cannot transition, missing description."
					return m, nil
				}

				if m.issueDetail.Children[m.childrenCursor].OriginalEstimate == "" {
					m.statusMessage = "Cannot transition, missing original estimate"
					return m, nil
				}

				m.mode = transitionView
				m.loadingTransitions = true
				m.transitionCursor = 0
				m.loadingDetail = true
				return m, m.fetchTransitions(m.issueDetail.Children[m.childrenCursor].Key)

			case keyPressMsg.String() == "e":
				m.mode = estimateView
				m.estimateData = NewEstimateFormData()
				return m, m.estimateData.Form.Init()

			case keyPressMsg.String() == "a":
				m.activeIssue = &m.issueDetail.Children[m.childrenCursor]
				m.mode = userSearchView
				m.loadingAssignUsers = true
				m.textInput.SetValue("")
				m.textInput.Focus()
				m.cursor = 0
				return m, m.fetchAssignableUsers(m.issueDetail.Children[m.childrenCursor].Key)
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
			m.priorityData = NewPriorityFormData(m.priorities, m.issueDetail.Priority.Name)
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
			w := &jira.Worklog{
				Time:        0,
				StartDate:   time.Now().Format("2006-01-02"),
				Description: "",
			}
			m.worklogFormData = m.NewWorklogForm(w, 40)
			m.mode = worklogView
			return m, m.worklogFormData.Form.Init()

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
			var cmds []tea.Cmd
			m.mode = listView
			m.issueDetail = nil
			m.childrenViewport.SetContent("")
			m.worklogsViewport.SetContent("")
			m.textArea.SetValue("")
			m.loadingIssues = true
			m.commentsCursor = 0
			m.worklogsCursor = 0
			m.childrenCursor = 0
			cmds = append(cmds, m.fetchMyIssues())
			return m, tea.Batch(cmds...)

		case "q", "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, cmd
}

func (m model) renderDetailView() string {
	if m.issueDetail == nil {
		return ui.PanelActiveSecondaryStyle.Render("Loading issue...")
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
