package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateListView(msg tea.Msg) (tea.Model, tea.Cmd) {

	if keyMsg, ok := msg.(tea.KeyMsg); ok {

		issuesToShow := m.issues
		if m.filterInput.Value() != "" {
			issuesToShow = filterIssues(m.issues, m.filterInput.Value())
		}

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
			if m.cursor > 0 {
				m.cursor--
				if m.cursor < len(issuesToShow) {
					return m, nil
				}
			}
		case "down", "j":
			if m.cursor < len(issuesToShow)-1 {
				m.cursor++
				if m.cursor < len(issuesToShow) {
					return m, nil
				}
			}
		case "/":
			m.filtering = true
			m.filterInput.SetValue("")
			m.filterInput.Focus()
			m.cursor = 0
			return m, textinput.Blink
		case "enter":
			if len(issuesToShow) > 0 && m.cursor < len(issuesToShow) {
				m.selectedIssue = &issuesToShow[m.cursor]
				m.mode = detailView
				m.loadingDetail = true
				m.loadingWorkLogs = true
				m.issueDetail = nil

				width := m.windowWidth - 10
				height := m.windowHeight - 15
				vp := viewport.New(width, height)
				m.detailViewport = &vp
				detailCmd := m.fetchIssueDetail(m.selectedIssue.Key)
				wlsCmd := m.fetchWorkLogs(m.selectedIssue.ID)

				return m, tea.Batch(detailCmd, wlsCmd)
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
	log.Printf("renderListView - windowWidth: %d, panelWidth: %d", m.windowWidth, panelWidth)
	panelHeight := m.windowHeight - 6

	issuesToShow := m.issues
	if m.filterInput.Value() != "" {
		issuesToShow = filterIssues(m.issues, m.filterInput.Value())
	}

	maxVisible := panelHeight - 2 // title and padding
	start := 0
	end := min(len(issuesToShow), maxVisible)

	if m.cursor >= end {
		start = m.cursor - maxVisible + 1
		end = m.cursor + 1
	} else if m.cursor < start {
		start = m.cursor
		end = start + maxVisible
	}

	start = max(0, start)
	end = min(len(issuesToShow), end)

	if start >= len(issuesToShow) {
		start = 0
		end = min(len(issuesToShow), maxVisible)
	}

	var listContent strings.Builder

	headers := ui.TypeHeader + ui.EmptyHeaderSpace +
		ui.KeyHeader +
		ui.PriorityHeader + ui.EmptyHeaderSpace +
		ui.SummaryHeader + ui.EmptyHeaderSpace + ui.EmptyHeaderSpace + // extra space for icon offset
		ui.StatusHeader + ui.EmptyHeaderSpace +
		ui.AssigneeHeader
	listContent.WriteString(headers)
	listContent.WriteString(ui.SeparatorStyle.Render(ui.RepeatChar("─", panelWidth-6)) + "\n")

	for i := start; i < end; i++ {
		issue := issuesToShow[i]

		issueType := ui.RenderIssueType(issue.Type)
		key := ui.KeyFieldStyle.Render(issue.Key)
		priority := ui.RenderPriority(issue.Priority)
		summary := ui.SummaryFieldStyle.Render(truncateLongString(issue.Summary, ui.ColWidthSummary))
		statusBadge := ui.RenderStatusBadge(issue.Status)
		assignee := ui.AssigneeFieldStyle.Render("@" + truncateLongString(issue.Assignee, 20))

		line := issueType + ui.EmptyHeaderSpace +
			key +
			priority + ui.EmptyHeaderSpace +
			summary + ui.EmptyHeaderSpace +
			statusBadge + ui.EmptyHeaderSpace +
			assignee

		if m.cursor == i {
			cursor := ui.IconCursor
			line = cursor + ui.SelectedRowStyle.Render(line)
		} else {
			line = "  " + ui.NormalRowStyle.Render(line)
		}

		listContent.WriteString(line + "\n")
	}

	var statusBar string
	if m.filtering {
		statusBar = ui.StatusBarKeyStyle.Render("Filter: ") + m.filterInput.View() +
			ui.StatusBarDescStyle.Render(" (enter to confirm, esc to cancel)")
	} else if m.filterInput.Value() != "" {
		statusBar = fmt.Sprintf("%s '%s' %s | %s | %s",
			ui.StatusBarDescStyle.Render("Filtered:"),
			ui.StatusBarKeyStyle.Render(m.filterInput.Value()),
			ui.StatusBarDescStyle.Render(fmt.Sprintf("(%d/%d)", len(issuesToShow), len(m.issues))),
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

	listPanel := ui.PanelStyleActive.
		Width(panelWidth).
		Height(panelHeight).
		Render(listContent.String())

	return listPanel + "\n" + ui.StatusBarStyle.Render(statusBar)
}
