package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
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
	).WithWidth(60)

	return d
}

func (m model) updateEditDescriptionView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
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
		m.loadingCount++
		cmds = append(cmds, m.updateDescriptionCmd(m.issueDetail.Key, description))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderEditDescriptionView() string {
	bg := lipgloss.NewLayer(m.renderDetailView())

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(m.descriptionData.Form.View())

	styledModal := ui.ModalBlockInputStyle.Render(modalContent.String())

	modalWidth := lipgloss.Width(styledModal)
	modalHeight := lipgloss.Height(styledModal)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
