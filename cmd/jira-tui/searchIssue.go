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

func NewSearchFormData() *SearchIssueFormData {
	e := &SearchIssueFormData{
		Query: "",
	}
	e.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Search Issue").
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
			m.searchData = nil
			return m, nil
		}
	}

	form, cmd := m.searchData.Form.Update(msg)

	if f, ok := form.(*huh.Form); ok {
		m.searchData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.searchData.Form.State == huh.StateCompleted {
		switch m.issueSelectionMode {
		case standardIssueSearch:
			m.loadingCount++
			cmds = append(cmds, m.fetchIssueDetailCmd(m.searchData.Query))
			m.statusMessage = statusMessage{
				msgType: infoStatusBarMsg,
				content: "Searching...",
			}
		case linkIssue:
			cmds = append(cmds, m.postLinkIssueCmd(m.issueDetail.Key, m.issueLinkData.IssueKey, m.issueLinkData.Relation))
			m.statusMessage = statusMessage{
				msgType: infoStatusBarMsg,
				content: "Linking...",
			}
		}
	}

	m.searchData.Err = nil

	return m, tea.Batch(cmds...)
}

func (m model) renderSearchIssueView() string {
	var bgContent string
	if m.issueSelectionMode == linkIssue {
		bgContent = m.renderDetailView()
	} else {
		bgContent = m.renderListView()
	}

	bg := lipgloss.NewLayer(bgContent)

	var modalContent strings.Builder

	modalContent.WriteString(m.searchData.Form.View())
	modalContent.WriteString("\n")

	if m.searchData.Err != nil {
		modalContent.WriteString(ui.ErrorStyle.Render("Error: " + m.searchData.Err.Error()))
	}

	styledModal := ui.ModalStyle.Render(modalContent.String())

	modalWidth := lipgloss.Width(styledModal)
	modalHeight := lipgloss.Height(styledModal)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
