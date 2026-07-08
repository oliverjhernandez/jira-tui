package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

type CommentFormData struct {
	Comment string
	Form    *huh.Form
}

func (m model) updateCommentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = detailView
			return m, nil

		case "@":
			var cmds []tea.Cmd
			m.mode = userSearchView
			m.userSelectionMode = insertMention

			if m.usersCache != nil {
				m.loadingCount++
				m.searchUserData = NewSearchUserFormData(m.usersCache)
				cmds = append(cmds, m.searchUserData.Form.Init())
			}

			return m, tea.Batch(cmds...)

		case "alt+enter", "ctrl+s":
			var cmd tea.Cmd
			comment := m.textArea.Value()
			if m.editingComment {
				cmd = m.updateCommentCmd(m.activeIssue.Key, m.activeIssue.Comments[m.commentsCursor].ID, comment)
				m.editingComment = false
				return m, cmd
			} else if m.textArea.Value() != "" {
				m.textArea.Reset()
				m.mode = detailView
				cmd = m.postCommentCmd(m.activeIssue.Key, comment)
				return m, cmd
			}
		}
	}

	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) renderCommentView() string {
	return m.renderModal("New Comment", m.textArea.View(), 0.3, 0.3)
}
