package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateListView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {

		// issuesToShow := m.issues
		// if m.filterInput.Value() != "" {
		// 	issuesToShow = filterIssues(m.issues, m.filterInput.Value())
		// }

		if m.filtering {
			switch keyMsg.String() {
			case "esc":
				m.filtering = false
				m.filterInput.SetValue("")
				m.filterInput.Blur()
				m.cursor = 0
				return m, nil
			case "enter":
				m.filtering = false
				m.filterInput.Blur()
				return m, nil
			}

			var cmd tea.Cmd
			m.filterInput, cmd = m.filterInput.Update(msg)

			return m, cmd
		}

		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit

		case "up", "k":
			if m.cursor == 0 {
				if m.sectionCursor > 0 {
					m.sectionCursor--
					m.cursor = len(m.sections[m.sectionCursor].Issues) - 1
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
			sectionIssues := m.sections[m.sectionCursor].Issues
			if m.cursor == len(sectionIssues)-1 {
				if m.sectionCursor < len(m.sections)-1 {
					m.sectionCursor++
					m.cursor = 0
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
			return m, nil
		case "/":
			m.filtering = true
			m.filterInput.SetValue("")
			m.filterInput.Focus()
			m.cursor = 0
			return m, textinput.Blink

		case "enter":
			if m.sectionCursor < len(m.sections) && m.cursor < len(m.sections[m.sectionCursor].Issues) {
				m.selectedIssue = m.sections[m.sectionCursor].Issues[m.cursor]
				m.loadingDetail = true
				m.loadingWorkLogs = true
				m.issueDetail = nil

				width := m.windowWidth - 10
				height := m.windowHeight - 15
				vp := viewport.New(width, height)
				m.detailViewport = &vp
				detailCmd := m.fetchIssueDetail(m.selectedIssue.Key)
				wlsCmd := m.fetchWorkLogs(m.selectedIssue.ID)

				return m, tea.Batch(detailCmd, wlsCmd, m.spinner.Tick)
			}

		case "esc":
			m.filterInput.SetValue("")
			m.cursor = 0
		}

	}

	return m, nil
}

func (m model) renderInfoPanel() string {
	panelWidth := m.getPanelWidth()

	userName := "loading..."
	if m.myself != nil {
		userName = "@" + m.myself.Name
	}

	var inProgress, toDo, done int
	for _, s := range m.sections {
		switch s.CategoryKey {
		case "indeterminate":
			inProgress = len(s.Issues)
		case "new":
			toDo = len(s.Issues)
		case "done":
			done = len(s.Issues)
		}
	}
	total := inProgress + toDo + done

	projectsStr := strings.Join(Projects, " · ")

	userStyled := ui.InfoPanelUserStyle.Render(userName)
	projectsStyled := ui.InfoPanelProjectStyle.Render(projectsStr)
	line1InnerWidth := panelWidth - 6
	line1Gap := line1InnerWidth - lipgloss.Width(userStyled) - lipgloss.Width(projectsStyled)
	if line1Gap < 0 {
		line1Gap = 1
	}
	line1 := userStyled + strings.Repeat(" ", line1Gap) + projectsStyled

	statusCounts := fmt.Sprintf("%s In Progress: %d    %s To Do: %d    %s Done: %d",
		ui.IconInfoInProgress, inProgress,
		ui.IconInfoToDo, toDo,
		ui.IconInfoDone, done)
	totalStr := ui.InfoPanelTotalStyle.Render(fmt.Sprintf("%d issues", total))
	line2Gap := line1InnerWidth - lipgloss.Width(statusCounts) - lipgloss.Width(totalStr)
	if line2Gap < 0 {
		line2Gap = 1
	}
	line2 := statusCounts + strings.Repeat(" ", line2Gap) + totalStr

	content := line1 + "\n" + line2
	return ui.InfoPanelStyle.Width(panelWidth).Render(content)
}

func (m model) renderListView() string {
	panelWidth := m.getPanelWidth()

	var listContent strings.Builder

	for si, s := range m.sections {
		paddingLeft := ui.SeparatorStyle.Render("───")
		sectionHeader := fmt.Sprintf("%s (%d) ", s.Name, len(s.Issues))
		paddingRight := ui.SeparatorStyle.Render(ui.RepeatChar("─", panelWidth-lipgloss.Width(sectionHeader)))
		fmt.Fprintf(&listContent, "%s%s%s\n", paddingLeft, sectionHeader, paddingRight)

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

	m.listViewport.SetContent(listContent.String())
	m.listViewport.YPosition = 0

	var statusBar string
	if m.filtering {
		statusBar = ui.StatusBarKeyStyle.Render("Filter: ") + m.filterInput.View() +
			ui.StatusBarDescStyle.Render(" (enter to confirm, esc to cancel)")
	} else if m.filterInput.Value() != "" {
		statusBar = fmt.Sprintf("%s '%s' %s | %s | %s",
			ui.StatusBarDescStyle.Render("Filtered:"),
			ui.StatusBarKeyStyle.Render(m.filterInput.Value()),
			ui.StatusBarDescStyle.Render(fmt.Sprintf("(%d/%d)", 10, len(m.issues))),
			ui.RenderKeyBind("/", "change"),
			ui.RenderKeyBind("esc", "clear"),
		)
	} else {
		statusBar = strings.Join([]string{
			ui.RenderKeyBind("/", "filter"),
			ui.RenderKeyBind("enter", "detail"),
			ui.RenderKeyBind("t", "transition"),
			ui.RenderKeyBind("q", "quit"),
		}, "  ")
	}

	infoPanel := m.renderInfoPanel()
	return infoPanel + "\n" + m.listViewport.View() + "\n" + ui.StatusBarStyle.Render(statusBar)
}
