package main

import (
	"log"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
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
			return m, nil
		}
	}

	form, cmd := m.searchUserData.Form.Update(msg)

	if f, ok := form.(*huh.Form); ok {
		m.searchUserData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.searchUserData.Form.State == huh.StateCompleted {
		log.Printf("Form completed, ID: %s", m.searchUserData.ID)
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
			cmds = append(cmds, m.postAssigneeCmd(m.activeIssue.Key, user.ID))
			if m.focusedSection == metadataSection {
				m.loadingCount++
				cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
			} else if m.focusedSection == subTasksSection {
				m.loadingCount++
				cmds = append(cmds, m.fetchSubTasksCmd(m.activeIssue.Key))
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
	log.Printf("Previeous Mode: %s", m.previousMode.String())
	var bg *lipgloss.Layer
	if m.previousMode == detailView {
		bg = lipgloss.NewLayer(m.renderDetailView())
	} else if m.previousMode == listView {
		bg = lipgloss.NewLayer(m.renderListView())
	}

	var modalContent strings.Builder

	modalContent.WriteString("\n")
	modalContent.WriteString(m.searchUserData.Form.View())

	if m.searchUserData.Err != nil {
		m.statusMessage = statusMessage{
			msgType: errStatusBarMsg,
			content: m.searchIssueData.Err.Error(),
		}
	}

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.2)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.1)

	var label string
	if m.userSelectionMode == assignUser && m.activeIssue != nil {
		label = "Assign " + m.activeIssue.Key
	} else {
		label = "Mention User"
	}
	styledModal := ui.RenderPanelWithLabel(label, modalContent.String(), modalWidth, modalHeight, true)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
