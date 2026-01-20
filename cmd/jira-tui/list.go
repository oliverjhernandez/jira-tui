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

func (m model) renderListView() string {
	panelWidth := max(120, m.windowWidth-4)

	var listContent strings.Builder

	// headers := ui.TypeHeader + ui.EmptyHeaderSpace +
	// 	ui.KeyHeader +
	// 	ui.PriorityHeader + ui.EmptyHeaderSpace +
	// 	ui.SummaryHeader + ui.EmptyHeaderSpace + ui.EmptyHeaderSpace +
	// 	ui.StatusHeader + ui.EmptyHeaderSpace +
	// 	ui.AssigneeHeader
	// separator := ui.SeparatorStyle.Render(ui.RepeatChar("─", panelWidth-6))
	// columnHeaders := headers

	for si, s := range m.sections {
		paddingLeft := ui.SeparatorStyle.Render("───")
		sectionHeader := fmt.Sprintf("%s (%d) ", s.Name, len(s.Issues))
		paddingRight := ui.SeparatorStyle.Render(ui.RepeatChar("─", panelWidth-lipgloss.Width(sectionHeader)))
		fmt.Fprintf(&listContent, "%s%s%s\n", paddingLeft, sectionHeader, paddingRight)

		for ii, issue := range s.Issues {
			issueType := ui.RenderIssueType(issue.Type, false)
			key := ui.KeyFieldStyle.Render(issue.Key)
			priority := ui.RenderPriority(issue.Priority, false)
			summary := ui.SummaryFieldStyle.Render(truncateLongString(issue.Summary, ui.ColWidthSummary))
			statusBadge := ui.RenderStatusBadge(issue.Status)
			assignee := ui.AssigneeFieldStyle.Render("@" + truncateLongString(issue.Assignee, 20))

			line := issueType + ui.EmptyHeaderSpace +
				key +
				priority + ui.EmptyHeaderSpace +
				summary + ui.EmptyHeaderSpace +
				statusBadge + ui.EmptyHeaderSpace +
				assignee

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

	return m.listViewport.View() + "\n" + ui.StatusBarStyle.Render(statusBar)
}
