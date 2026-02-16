package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type DescriptionFormData struct {
	Description string
	Form        *huh.Form
}

func NewDescriptionFormData(initialValue string) *DescriptionFormData {
	d := &DescriptionFormData{
		Description: initialValue,
	}
	d.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Description").
				Value(&d.Description).
				Lines(15),
		),
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(60)

	return d
}

func (m model) updateEditDescriptionView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			m.editingDescription = false
			return m, m.descriptionData.Form.Init()
		}
	}

	form, cmd := m.descriptionData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.descriptionData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.descriptionData.Form.State == huh.StateCompleted {
		m.mode = detailView
		m.editingDescription = false
		description := m.descriptionData.Description
		cmds = append(cmds, m.updateDescription(m.issueDetail.Key, description))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderEditDescriptionView() string {
	bg := m.renderSimpleBackground()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(m.descriptionData.Form.View())

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowWidth, m.windowHeight, ui.ModalBlockInputStyle)
}
