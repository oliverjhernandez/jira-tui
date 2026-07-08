package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
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
		cmds = append(cmds, m.updateDescriptionCmd(m.activeIssue.Key, description))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderEditDescriptionView() string {
	return m.renderModal("Description", m.descriptionData.Form.View(), 0.3, 0.3)
}
