package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

type SearchUserFormData struct {
	ID   string
	Form *huh.Form
	Err  error
}

func NewSearchUserFormData(users []jira.User) *SearchUserFormData {
	options := make([]huh.Option[string], len(users))
	for i, u := range users {
		options[i] = huh.NewOption(u.Name, u.ID)
	}

	uf := &SearchUserFormData{
		ID: "",
	}
	uf.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Options(options...).
				Value(&uf.ID).
				Filtering(true).
				Height(1),
		),
	).WithWidth(50)

	return uf
}

func (m model) updateSearchUserView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = m.previousMode
			m.searchUserData = nil
			m.userSelectionMode = 0
			return m, nil
		}
	}

	form, cmd := m.searchUserData.Form.Update(msg)

	if f, ok := form.(*huh.Form); ok {
		m.searchUserData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.searchUserData.Form.State == huh.StateCompleted {
		var user jira.User
		for _, u := range m.usersCache {
			if u.ID == m.searchUserData.ID {
				user = u
			}
		}
		switch m.userSelectionMode {
		case insertMention:
			mention := "@[" + user.Name + "]"
			value := m.textArea.Value() + mention
			m.textArea.SetValue(value)
			m.mode = commentView
			return m, nil

		case assignUser:
			if m.pendingIssue == nil {
				m.mode = m.previousMode
				m.searchUserData = nil
				return m, nil
			}
			cmds = append(cmds, m.postAssigneeCmd(m.pendingIssue.Key, user.ID))
			if m.focusedSection == metadataSection {
				m.loadingCount++
				cmds = append(cmds, m.fetchIssueDetailCmd(m.pendingIssue.Key))
			} else if m.focusedSection == subTasksSection {
				m.loadingCount++
				cmds = append(cmds, m.fetchSubTasksCmd(m.pendingIssue.Key))
			}
			m.loadingCount++
			cmds = append(cmds, m.fetchMyIssuesCmd())

			m.mode = m.previousMode
			m.searchUserData = nil
			return m, tea.Batch(cmds...)
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderSearchUserView() string {
	var modalContent strings.Builder

	modalContent.WriteString("\n")
	modalContent.WriteString(m.searchUserData.Form.View())

	if m.searchUserData.Err != nil {
		m.statusMessage = statusMessage{
			msgType: errStatusBarMsg,
			content: m.searchUserData.Err.Error(),
		}
	}

	var label string
	if m.userSelectionMode == assignUser && m.pendingIssue != nil {
		label = "Assign " + m.pendingIssue.Key
	} else {
		label = "Mention User"
	}

	return m.renderModal(label, modalContent.String(), 0.2, 0.1)
}
