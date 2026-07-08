package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
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
			if m.pendingIssue != nil && m.issueLinkData != nil {
				cmds = append(cmds, m.postLinkIssueCmd(m.pendingIssue.Key, m.issueLinkData.IssueKey, m.issueLinkData.Relation))
				m.statusMessage = statusMessage{
					msgType: infoStatusBarMsg,
					content: "Linking...",
				}
			}
		}
	}

	m.searchIssueData.Err = nil

	return m, tea.Batch(cmds...)
}

func (m model) renderSearchIssueView() string {
	var modalContent strings.Builder

	modalContent.WriteString(m.searchIssueData.Form.View())
	modalContent.WriteString("\n")

	if m.searchIssueData.Err != nil {
		m.statusMessage = statusMessage{
			msgType: errStatusBarMsg,
			content: m.searchIssueData.Err.Error(),
		}
	}

	return m.renderModal("Search Issue", modalContent.String(), 0.2, 0.1)
}
