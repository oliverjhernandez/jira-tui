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

func NewCommentFormData() *CommentFormData {
	c := &CommentFormData{
		Comment: "",
	}
	c.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Comment").
				Value(&c.Comment).
				Lines(10),
		),
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(60)

	return c
}

func (m model) updatePostCommentView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			return m, m.commentData.Form.Init()

		case "@":
			m.mode = userSearchView
			m.userSelectionMode = insertMention
			m.loadingAssignUsers = true
			m.textInput.SetValue("")
			m.textInput.Focus()
			m.cursor = 0
			return m, m.fetchUsers(m.issueDetail.Key)

		case "alt+enter", "ctrl+s":
			if m.textArea.Value() != "" {
				comment := m.textArea.Value()
				m.textArea.Reset()
				m.mode = detailView
				return m, m.postComment(m.issueDetail.Key, comment)
			}
			return m, nil
		}
	}

	if m.commentData.Form.State == huh.StateCompleted {
		m.mode = detailView
		comment := m.commentData.Comment
		cmds = append(cmds, m.postComment(m.issueDetail.Key, comment))
	}

	m.textArea, cmd = m.textArea.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m model) renderPostCommentView() string {
	m.detailLayout = m.calculateDetailLayout()
	bg := m.renderSimpleBackground()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(m.textArea.View())

	footer := strings.Join([]string{
		ui.RenderKeyBind("ctrl+enter", "submit"),
		ui.RenderKeyBind("esc", "cancel"),
	}, "  ")
	modalContent.WriteString("\n" + footer)

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowWidth, m.windowHeight, ui.ModalBlockInputStyle)
}
