package main

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			m.postingComment = false
			return m, m.commentData.Form.Init()
		}
	}

	form, cmd := m.commentData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.commentData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.commentData.Form.State == huh.StateCompleted {
		m.mode = detailView
		m.postingComment = false
		comment := m.commentData.Comment
		cmds = append(cmds, m.postComment(m.issueDetail.Key, comment))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderPostCommentView() string {
	log.Printf("=== renderPostCommentView called ===")

	bg := m.renderDetailView()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalWidth := m.getLargeModalWidth()
	modalHeight := m.getModalHeight(0.6)

	modalContent.WriteString(m.commentData.Form.View())

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight).
		Background(lipgloss.Color("235"))

	styledModal := modalStyle.Render(modalContent.String())
	overlay := PlaceOverlay(10, 20, styledModal, bg, false)

	return overlay
}
