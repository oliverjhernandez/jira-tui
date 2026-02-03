package main

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateAssignUsersView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.statusBarInput.SetValue("")
			m.statusBarInput.Blur()
			m.cursor = 0
			m.mode = detailView
			return m, nil
		case "enter":
			m.filtering = false
			m.statusBarInput.SetValue("")
			m.statusBarInput.Blur()
			m.mode = detailView

			if len(m.filteredUsers) > 0 {
				assignee := m.filteredUsers[m.assigneeCursor]
				assigneeCmd := m.postAssignee(m.issueDetail.Key, assignee.ID)
				detailCmd := m.fetchIssueDetail(m.issueDetail.Key)
				listCmd := m.fetchMyIssues()
				m.filteredUsers = nil
				return m, tea.Batch(assigneeCmd, detailCmd, listCmd)
			}
		}

		var cmd tea.Cmd
		m.statusBarInput, cmd = m.statusBarInput.Update(msg)

		m.filteredUsers = nil
		match := m.statusBarInput.Value()
		if len(match) >= 2 {
			for _, u := range m.assignUsersCache {
				if strings.HasPrefix(strings.ToLower(u.Name), strings.ToLower(match)) {
					m.filteredUsers = append(m.filteredUsers, &u)
				}
			}
		}

		return m, cmd
	}

	return m, nil
}

func (m model) renderAssignUsersView() string {
	log.Printf("=== renderAssignUsersView called ===")

	bg := m.renderDetailView()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		fmt.Fprintf(&modalContent, "Change Assignee for %s\n", m.issueDetail.Key)
	}

	if m.loadingAssignUsers {
		modalContent.WriteString(m.spinner.View() + "Loading available users...\n")
	} else if len(m.assignUsersCache) == 0 {
		modalContent.WriteString("No assignable users for this issue.\n")
	}

	modalContent.WriteString(m.statusBarInput.View() + "\n\n")

	for i, u := range m.filteredUsers {
		if i == m.assigneeCursor {
			modalContent.WriteString("> " + u.Name + "\n")
		} else {
			modalContent.WriteString("  " + u.Name + "\n")
		}
	}

	footer := strings.Join([]string{
		ui.RenderKeyBind("type", "search"),
		ui.RenderKeyBind("enter", "select"),
		ui.RenderKeyBind("esc", "cancel"),
	}, "  ")
	modalContent.WriteString("\n" + footer)

	return ui.RenderCenteredModal(
		modalContent.String(),
		bg,
		m.windowWidth,
		m.windowHeight,
		ui.ModalTextInputStyle,
	)
}
