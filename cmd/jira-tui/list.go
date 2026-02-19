package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateListView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {

		if m.filtering {
			switch keyMsg.String() {
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
		case keyMsg.String() == "g" && m.lastKey == "":
			m.lastKey = "g"
			tick := tea.Tick(300*time.Millisecond, func(t time.Time) tea.Msg {
				return keyTimeoutMsg{}
			})
			return m, tick
		case keyMsg.String() == "g" && m.lastKey == "g":
			m.lastKey = ""
			m.cursor = 0
			m.sectionCursor = 0
			m.listViewport.GotoTop()
		default:
			m.lastKey = ""
		}

		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			sectionsToNavigate := m.sections
			if m.filteredSections != nil {
				sectionsToNavigate = m.filteredSections
			}

			if m.cursor == 0 {
				for prevSection := m.sectionCursor - 1; prevSection >= 0; prevSection-- {
					if len(sectionsToNavigate[prevSection].Issues) > 0 {
						m.sectionCursor = prevSection
						m.cursor = len(sectionsToNavigate[prevSection].Issues) - 1
						break
					}
				}
			} else {
				m.cursor--
			}

			cursorLine := m.getAbsoluteCursorLine()
			viewportHeight := m.listViewport.Height
			currentOffset := m.listViewport.YOffset

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
			if m.cursor == len(sectionIssues)-1 {
				for nextSection := m.sectionCursor + 1; nextSection < len(sectionsToNavigate); nextSection++ {
					if len(sectionsToNavigate[nextSection].Issues) > 0 {
						m.sectionCursor = nextSection
						m.cursor = 0
						break
					}
				}
			} else {
				m.cursor++
			}

			cursorLine := m.getAbsoluteCursorLine()
			viewportHeight := m.listViewport.Height

			if cursorLine >= m.listViewport.YOffset+viewportHeight {
				m.listViewport.SetYOffset(cursorLine - viewportHeight + 1)
			}
			if cursorLine < m.listViewport.YOffset {
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
			if m.loading {
				return m, nil
			}
			m.loading = true
			return m, m.fetchMyIssues()

		case "enter":
			sectionsToNavigate := m.sections
			if m.filteredSections != nil {
				sectionsToNavigate = m.filteredSections
			}

			if m.sectionCursor < len(sectionsToNavigate) && m.cursor < len(sectionsToNavigate[m.sectionCursor].Issues) {
				m.selectedIssue = sectionsToNavigate[m.sectionCursor].Issues[m.cursor]
				m.loadingDetail = true
				m.loadingWorkLogs = true
				m.issueDetail = nil

				var cmds []tea.Cmd

				detailCmd := m.fetchIssueDetail(m.selectedIssue.Key)
				worklogsCmd := m.fetchWorkLogs(m.selectedIssue.ID)

				cmds = append(cmds, detailCmd)
				cmds = append(cmds, worklogsCmd)
				cmds = append(cmds, m.spinner.Tick)

				if m.selectedIssue.Type == "Epic" {
					epicChildrenCmd := m.fetchEpicChildren(m.selectedIssue.Key)
					cmds = append(cmds, epicChildrenCmd)
				}

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
			statusBadge := ui.RenderStatusBadge(issue.Status)
			assignee := m.columnWidths.RenderAssignee("@" + truncateLongString(issue.Assignee, m.columnWidths.Assignee))
			worklogSeconds := m.worklogTotals[issue.ID]
			timeSpent := m.columnWidths.RenderTimeSpent(ui.FormatTimeSpent(worklogSeconds))

			emptySpace := m.columnWidths.RenderEmptySpace()
			line := issueType + emptySpace +
				key +
				priority + emptySpace +
				summary + emptySpace +
				statusBadge + emptySpace +
				assignee + emptySpace +
				timeSpent

			if m.sectionCursor == si && m.cursor == ii {
				cursor := ui.IconCursor

				if m.loadingDetail {
					line = m.spinner.View() + " " + ui.SelectedRowStyle.Render(line)
				} else {
					line = cursor + ui.SelectedRowStyle.Render(line)
				}
			} else {
				line = "  " + ui.NormalRowStyle.Render(line)
			}

			listContent.WriteString(line + "\n")
		}
	}

	panelsHeight := 6 + // infoPanel height
		4 + // horizontal borders
		1 // statusBar height
	m.listViewport.Height = m.windowHeight - panelsHeight
	m.listViewport.Width = ui.GetAvailableWidth(m.windowWidth) - 4
	m.listViewport.SetContent(listContent.String())
	m.listViewport.YPosition = 0

	var statusBar strings.Builder
	if m.filtering {
		statusBar.WriteString(ui.StatusBarKeyStyle.Render("Filter: ") + m.textInput.View() +
			ui.StatusBarDescStyle.Render(" (enter to confirm, esc to cancel)"))
	} else if m.textInput.Value() != "" {
		fmt.Fprintf(&statusBar, "%s '%s' %s | %s | %s",
			ui.StatusBarDescStyle.Render("Filtered:"),
			ui.StatusBarKeyStyle.Render(m.textInput.Value()),
			ui.StatusBarDescStyle.Render(fmt.Sprintf("(%d/%d)", 10, len(m.issues))),
			ui.RenderKeyBind("/", "change"),
			ui.RenderKeyBind("esc", "clear"))
	} else {
		statusBar.WriteString(strings.Join([]string{
			ui.RenderKeyBind("/", "filter"),
			ui.RenderKeyBind("enter", "detail"),
			ui.RenderKeyBind("t", "transition"),
			ui.RenderKeyBind("q", "quit"),
		}, "  "))
	}

	panelWidth := ui.GetAvailableWidth(m.windowWidth)
	infoPanel := m.renderInfoPanel(panelWidth)
	return infoPanel + "\n" + ui.PanelActiveStyle.Render(m.listViewport.View()) + "\n" + ui.StatusBarStyle.Render(statusBar.String())
}
