package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type CommentFormData struct {
	Comment string
	Form    *huh.Form
}

func (m model) updateCommentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			return m, nil

		case "@":
			m.mode = userSearchView
			m.userSelectionMode = insertMention
			m.loadingAssignUsers = true
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.cursor = 0
			return m, m.fetchUsers(m.issueDetail.Key)

		case "alt+enter", "ctrl+s":
			var cmd tea.Cmd
			comment := m.textArea.Value()
			if m.editingComment {
				cmd = m.updateComment(m.issueDetail.Key, m.issueDetail.Comments[m.commentsCursor].ID, comment)
				return m, cmd
			} else if m.textArea.Value() != "" {
				m.textArea.Reset()
				m.mode = detailView
				cmd = m.postComment(m.issueDetail.Key, comment)
				return m, cmd
			}
		}
	}

	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) renderCommentModalView() string {
	m.detailLayout = m.calculateDetailLayout()
	bg := m.renderSimpleBackground()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n")
	}

	modalContent.WriteString(m.textArea.View())

	footer := strings.Join([]string{
		ui.RenderKeyBind("ctrl+enter", "submit"),
		ui.RenderKeyBind("esc", "cancel"),
	}, "  ")
	modalContent.WriteString("\n" + footer)

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowWidth, m.windowHeight, ui.ModalBlockInputStyle)
}
