package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateSearchUserView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.textInput.SetValue("")
			m.textInput.Blur()
			m.cursor = 0
			m.mode = detailView
			return m, nil
		case "enter":
			m.filtering = false
			m.textInput.SetValue("")
			m.textInput.Blur()
			m.mode = detailView

			if len(m.filteredUsers) > 0 {
				user := m.filteredUsers[m.userCursor]
				var cmds []tea.Cmd

				switch m.userSelectionMode {
				case assignUser:
					cmds = append(cmds, m.postAssigneeCmd(m.activeIssue.Key, user.ID))
					if m.focusedSection == metadataSection {
						m.loadingCount++
						cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
					} else if m.focusedSection == subTasksSection {
						m.loadingCount++
						cmds = append(cmds, m.fetchSubTasksCmd(m.issueDetail.Key))
					}
					m.loadingCount++
					cmds = append(cmds, m.fetchMyIssuesCmd())
					m.filteredUsers = nil
					return m, tea.Batch(cmds...)
				case insertMention:
					mention := "@[" + user.Name + "]"
					value := m.textArea.Value() + mention
					m.textArea.SetValue(value)
					m.mode = commentView
					return m, nil
				}
			}
		}

		var cmd tea.Cmd
		m.textInput, cmd = m.textInput.Update(msg)

		m.filteredUsers = nil
		match := m.textInput.Value()
		if len(match) >= 2 {
			for _, u := range m.usersCache {
				if strings.HasPrefix(strings.ToLower(u.Name), strings.ToLower(match)) {
					m.filteredUsers = append(m.filteredUsers, &u)
				}
			}
		}

		return m, cmd
	}

	return m, nil
}

func (m model) renderSearchUserView() string {
	bg := lipgloss.NewLayer(m.renderDetailView())

	var modalContent strings.Builder

	if m.issueDetail != nil {
		fmt.Fprintf(&modalContent, "Change Assignee for %s\n", m.issueDetail.Key)
	}

	modalContent.WriteString(m.textInput.View() + "\n\n")

	for i, u := range m.filteredUsers {
		if i == m.userCursor {
			modalContent.WriteString("> " + u.Name + "\n")
		} else {
			modalContent.WriteString("  " + u.Name + "\n")
		}
	}

	styledModal := ui.ModalBlockInputStyle.Render(modalContent.String())

	modalWidth := lipgloss.Width(styledModal)
	modalHeight := lipgloss.Height(styledModal)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
