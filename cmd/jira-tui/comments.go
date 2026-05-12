package main

import (
	"log"
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
	bg := lipgloss.NewLayer(m.renderDetailView())

	var modalContent strings.Builder

	modalContent.WriteString(m.textArea.View())

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.3)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.3)
	log.Printf("Modal Height: %d", modalHeight)
	log.Printf("Modal Width: %d", modalWidth)

	styledModal := ui.RenderPanelWithLabel("New Comment", modalContent.String(), modalWidth, modalHeight, true)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
