package main

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
				detailCmd := m.fetchIssueDetail(m.selectedIssue.Key)
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
	fmt.Fprintf(&modalContent, "Change Assignee for %s\n", m.selectedIssue.Key)
	modalContent.WriteString(strings.Repeat("=", 50) + "\n\n")

	if m.loadingAssignUsers {
		modalContent.WriteString(m.spinner.View() + "Loading available users...\n")
	} else if len(m.assignUsersCache) == 0 {
		modalContent.WriteString("No assignable users for this issue.\n")
	}

	modalWidth := m.getLargeModalWidth()
	modalHeight := 1

	modalContent.WriteString(m.statusBarInput.View() + "\n\n")

	log.Printf("Filtered List: %+v", m.filteredUsers)

	for i, u := range m.filteredUsers {
		modalHeight = 5 + len(m.filteredUsers)
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

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight).
		Background(lipgloss.Color("235"))

	styledModal := modalStyle.Render(modalContent.String())
	overlay := PlaceOverlay(10, 10, styledModal, bg, false)

	return overlay
}
