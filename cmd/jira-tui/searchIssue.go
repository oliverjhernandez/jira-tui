package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
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
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(40)

	return e
}

func (m model) updateSearchIssueView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
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
			m.loadingDetail = true
			cmds = append(cmds, m.fetchIssueDetail(m.searchData.Query))
		case linkIssue:
			m.loadingDetail = true
			if m.searchData.Query == jira.MonthlyChangeIssue {
				m.issueDetail.IsLinkedToChange = true
			}

			if m.issueDetail.IsLinkedToChange {
				cmds = append(cmds, m.unlinkIssue(m.issueDetail.ChangeIssueLinkID))
			} else {
				cmds = append(cmds, m.linkIssue(m.issueDetail.Key, jira.MonthlyChangeIssue))
			}
		}
	}

	m.searchData.Err = nil

	return m, tea.Batch(cmds...)
}

func (m model) renderSearchIssueView() string {
	var bg string
	if m.issueSelectionMode == linkIssue {
		bg = m.renderSimpleBackground()
	} else {
		bg = m.renderSimpleBackground()
	}

	if m.searchData.Form.State == huh.StateCompleted {
		return m.renderDetailView()
	}

	var modalContent strings.Builder

	modalContent.WriteString(m.searchData.Form.View())
	modalContent.WriteString("\n")

	if m.searchData.Err != nil {
		modalContent.WriteString(ui.ErrorStyle.Render("Error: " + m.searchData.Err.Error()))
	}

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowWidth, m.windowHeight, ui.ModalTextInputStyle)
}
