package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
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
			m.mode = userSearchView
			m.userSelectionMode = insertMention
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.cursor = 0
			m.loadingCount++
			return m, m.fetchAssignableUsersCmd(m.issueDetail.Key)

		case "alt+enter", "ctrl+s":
			var cmd tea.Cmd
			comment := m.textArea.Value()
			if m.editingComment {
				cmd = m.updateCommentCmd(m.issueDetail.Key, m.issueDetail.Comments[m.commentsCursor].ID, comment)
				m.editingComment = false
				return m, cmd
			} else if m.textArea.Value() != "" {
				m.textArea.Reset()
				m.mode = detailView
				cmd = m.postCommentCmd(m.issueDetail.Key, comment)
				return m, cmd
			}
		}
	}

	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) renderCommentView() string {
	bg := lipgloss.NewLayer(m.renderDetailView())

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n")
	}

	modalContent.WriteString(m.textArea.View())

	styledModal := ui.ModalStyle.Render(modalContent.String())

	modalWidth := lipgloss.Width(styledModal)
	modalHeight := lipgloss.Height(styledModal)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
