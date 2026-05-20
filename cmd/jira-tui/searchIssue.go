package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type SearchIssueFormData struct {
	Query string
	Form  *huh.Form
	Err   error
}

func NewSearchIssueFormData() *SearchIssueFormData {
	e := &SearchIssueFormData{
		Query: "",
	}
	e.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Placeholder("DEV-123").
				Value(&e.Query),
		),
	).WithWidth(40)

	return e
}

func (m model) updateSearchIssueView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = listView
			m.searchIssueData = nil
			return m, nil
		}
	}

	form, cmd := m.searchIssueData.Form.Update(msg)

	if f, ok := form.(*huh.Form); ok {
		m.searchIssueData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.searchIssueData.Form.State == huh.StateCompleted {
		switch m.issueSelectionMode {
		case standardIssueSearch:
			m.loadingCount++
			cmds = append(cmds, m.fetchIssueDetailCmd(m.searchIssueData.Query))
			m.statusMessage = statusMessage{
				msgType: infoStatusBarMsg,
				content: "Searching...",
			}
		case linkIssue:
			cmds = append(cmds, m.postLinkIssueCmd(m.activeIssue.Key, m.issueLinkData.IssueKey, m.issueLinkData.Relation))
			m.statusMessage = statusMessage{
				msgType: infoStatusBarMsg,
				content: "Linking...",
			}
		}
	}

	m.searchIssueData.Err = nil

	return m, tea.Batch(cmds...)
}

func (m model) renderSearchIssueView() string {
	var bgContent string
	// FIX: Shouldnt be based on mode
	if m.issueSelectionMode == linkIssue {
		bgContent = m.renderDetailView()
	} else {
		bgContent = m.renderListView()
	}

	bg := lipgloss.NewLayer(bgContent)

	var modalContent strings.Builder

	modalContent.WriteString(m.searchIssueData.Form.View())
	modalContent.WriteString("\n")

	if m.searchIssueData.Err != nil {
		m.statusMessage = statusMessage{
			msgType: errStatusBarMsg,
			content: m.searchIssueData.Err.Error(),
		}
	}

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.2)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.1)

	styledModal := ui.RenderPanelWithLabel("Search Issue", modalContent.String(), modalWidth, modalHeight, true)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
