package main

import (
	"fmt"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateListView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {

		if m.filtering {
			switch keyPressMsg.String() {
			case "esc":
				m.filtering = false
				m.textInput.SetValue("")
				m.textInput.Blur()
				m.cursor = 0
				m.sectionCursor = 0
				m.filteredSections = nil
				return m, nil
			case "enter":
				m.filtering = false
				m.textInput.Blur()
				m.cursor = 0
				m.sectionCursor = 0
				m.commentsCursor = 0
				for i, s := range m.filteredSections {
					if len(s.Issues) > 0 {
						m.sectionCursor = i
						m.cursor = 0
						break
					}
				}
				return m, nil
			}

			var cmd tea.Cmd
			m.textInput, cmd = m.textInput.Update(msg)

			if m.textInput.Value() != "" {
				m.filteredSections = filterSections(m.sections, m.textInput.Value())

				for i, s := range m.filteredSections {
					if len(s.Issues) > 0 {
						m.sectionCursor = i
						m.cursor = 0
						break
					}
				}

			} else {
				m.filteredSections = nil
			}

			return m, cmd
		}

		// sequential keybindings
		switch {
		case keyPressMsg.String() == "g" && m.lastKey == "":
			m.lastKey = "g"
			return m, nil

		case keyPressMsg.String() == "g" && m.lastKey == "g":
			m.lastKey = ""
			m.cursor = 0
			m.sectionCursor = 0
			m.listViewport.GotoTop()

		case keyPressMsg.String() == "y" && m.lastKey == "":
			m.lastKey = "y"
			return m, nil

		case keyPressMsg.String() == "k" && m.lastKey == "y":
			var cmds []tea.Cmd
			m.lastKey = ""
			textToCopy := m.sections[m.sectionCursor].Issues[m.cursor].Key
			yankToClipboard(textToCopy)
			m.statusMessage.content = "Key yanked to clipboard"
			cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
			return m, tea.Batch(cmds...)

		case keyPressMsg.String() == "K" && m.lastKey == "y":
			var cmds []tea.Cmd
			m.lastKey = ""
			textToCopy := "https://layer7.atlassian.net/browse/" + m.sections[m.sectionCursor].Issues[m.cursor].Key
			yankToClipboard(textToCopy)
			m.statusMessage.content = "URL yanked to clipboard"
			cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
			return m, tea.Batch(cmds...)

		case keyPressMsg.String() == "s" && m.lastKey == "y":
			var cmds []tea.Cmd
			m.lastKey = ""
			textToCopy := m.sections[m.sectionCursor].Issues[m.cursor].Summary
			yankToClipboard(textToCopy)
			m.statusMessage.content = "Summary yanked to clipboard"
			cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
			return m, tea.Batch(cmds...)
		}

		switch keyPressMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "n":
			i := &NewIssueFormData{}
			m.issueDetail = nil
			m.newIssueData = m.NewIssueForm(i)
			m.mode = newIssueView
			return m, m.newIssueData.Form.Init()

		case "up", "k":
			sectionsToNavigate := m.sections
			if m.filteredSections != nil {
				sectionsToNavigate = m.filteredSections
			}

			sectionIssues := sectionsToNavigate[m.sectionCursor].Issues
			if m.cursor == 0 || len(sectionIssues) == 0 {
				for prevSection := m.sectionCursor - 1; prevSection >= 0; prevSection-- {
					if len(sectionsToNavigate[prevSection].Issues) > 0 {
						m.sectionCursor = prevSection
						m.cursor = len(sectionsToNavigate[prevSection].Issues) - 1
						m.selectedIssue = sectionsToNavigate[prevSection].Issues[m.cursor]
						break
					}
				}
			} else {
				m.cursor--
				m.selectedIssue = sectionsToNavigate[m.sectionCursor].Issues[m.cursor]
			}

			cursorLine := m.getAbsoluteCursorLine()
			viewportHeight := m.listViewport.Height()
			currentOffset := m.listViewport.YOffset()

			topThreshold := currentOffset + (viewportHeight / 3)

			if cursorLine < topThreshold {
				newOffset := cursorLine - (viewportHeight / 3)
				m.listViewport.SetYOffset(max(0, newOffset))
			}

			return m, nil

		case "down", "j":
			sectionsToNavigate := m.sections
			if m.filteredSections != nil {
				sectionsToNavigate = m.filteredSections
			}

			sectionIssues := sectionsToNavigate[m.sectionCursor].Issues
			if m.cursor == len(sectionIssues)-1 || len(sectionIssues) == 0 {
				for nextSection := m.sectionCursor + 1; nextSection < len(sectionsToNavigate); nextSection++ {
					if len(sectionsToNavigate[nextSection].Issues) > 0 {
						m.sectionCursor = nextSection
						m.cursor = 0
						m.selectedIssue = sectionsToNavigate[nextSection].Issues[m.cursor]
						break
					}
				}
			} else {
				m.cursor++
				m.selectedIssue = sectionsToNavigate[m.sectionCursor].Issues[m.cursor]
			}

			cursorLine := m.getAbsoluteCursorLine()
			viewportHeight := m.listViewport.Height()
			currentOffset := m.listViewport.YOffset()

			if cursorLine >= currentOffset+viewportHeight {
				m.listViewport.SetYOffset(cursorLine - viewportHeight + 1)
			}
			if cursorLine < currentOffset {
				m.listViewport.SetYOffset(cursorLine)
			}
			return m, nil

		case "G":
			lenSection := len(m.sections) - 1
			lenIssues := len(m.sections[lenSection].Issues) - 1
			m.cursor = lenIssues
			m.sectionCursor = lenSection
			m.listViewport.GotoBottom()
			return m, nil

		case "ctrl+p":
			sortSectionsByPriority(m.sections)
			return m, nil

		case "ctrl+s":
			m.mode = issueSearchView
			m.searchData = NewSearchFormData()
			return m, m.searchData.Form.Init()

		case "/":
			m.filtering = true
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.cursor = 0
			m.sectionCursor = 0
			return m, textinput.Blink

		case "ctrl+r":
			if m.loadingCount > 0 {
				return m, nil
			}
			m.loadingCount++
			return m, m.fetchMyIssuesCmd()

		case "enter":
			m.activeIssue = m.selectedIssue
			sectionsToNavigate := m.sections
			if m.filteredSections != nil {
				sectionsToNavigate = m.filteredSections
			}

			if m.sectionCursor < len(sectionsToNavigate) && m.cursor < len(sectionsToNavigate[m.sectionCursor].Issues) {
				m.selectedIssue = sectionsToNavigate[m.sectionCursor].Issues[m.cursor]
				m.issueDetail = nil

				var cmds []tea.Cmd

				m.loadingCount++
				detailCmd := m.fetchIssueDetailCmd(m.selectedIssue.Key)
				cmds = append(cmds, detailCmd)

				return m, tea.Batch(cmds...)
			}

		case "esc":
			m.textInput.SetValue("")
			m.filteredSections = nil
			m.cursor = 0
			m.sectionCursor = 0
		}
	}

	return m, nil
}

func (m model) renderListView() string {
	var listContent strings.Builder

	sectionsToRender := m.sections
	if m.filteredSections != nil {
		sectionsToRender = m.filteredSections
	}

	for si, s := range sectionsToRender {
		sectionHeader := ui.SectionTitleStyle.Render(fmt.Sprintf("%s (%d)", s.Name, len(s.Issues)))
		fmt.Fprintf(&listContent, "%s\n", sectionHeader)

		for ii, issue := range s.Issues {
			issueType := ui.RenderIssueType(issue.Type, false)
			key := m.columnWidths.RenderKey(issue.Key)
			priority := ui.RenderPriority(issue.Priority, false)
			summary := m.columnWidths.RenderSummary(truncateLongString(issue.Summary, m.columnWidths.Summary))
			reporter := m.columnWidths.RenderReporter("@" + truncateLongString(issue.Reporter.ID, m.columnWidths.Assignee))
			statusBadge := ui.RenderStatusBadge(issue.Status)
			assignee := m.columnWidths.RenderAssignee("@" + truncateLongString(issue.Assignee, m.columnWidths.Assignee))
			worklogSeconds := m.worklogTotals[issue.ID]
			timeSpent := m.columnWidths.RenderTimeSpent(ui.FormatTimeSpent(worklogSeconds))

			emptySpace := m.columnWidths.RenderEmptySpace()
			line := issueType + emptySpace +
				key +
				priority + emptySpace +
				summary + emptySpace +
				reporter + emptySpace +
				statusBadge + emptySpace +
				assignee + emptySpace +
				timeSpent

			if m.sectionCursor == si && m.cursor == ii {
				cursor := ui.IconCursor
				line = cursor + ui.SelectedRowStyle.Render(line)
			} else {
				line = "  " + ui.NormalRowStyle.Render(line)
			}

			listContent.WriteString(line + "\n")
		}

		listContent.WriteString("\n\n")
	}

	m.listViewport.SetContent(listContent.String())
	m.listViewport.YPosition = 0

	var statusBar strings.Builder

	statusBar.WriteString(m.renderStatusBar())

	infoPanel := m.renderInfoPanel()
	return infoPanel + "\n" + ui.PanelActiveStyle.Render(m.listViewport.View()) + "\n" + statusBar.String()
}
