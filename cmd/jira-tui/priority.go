package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

type PriorityFormData struct {
	SelectedPriority string
	Form             *huh.Form
}

func NewPriorityFormData(priorities []jira.Priority, current string) *PriorityFormData {
	options := make([]huh.Option[string], len(priorities))
	for i, p := range priorities {
		options[i] = huh.NewOption(p.Name, p.Name)
	}

	p := &PriorityFormData{
		SelectedPriority: current,
	}
	p.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Priority").
				Options(options...).
				Value(&p.SelectedPriority),
		),
	).WithWidth(30)

	return p
}

func (m model) updateEditPriorityView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = m.previousMode
			return m, m.priorityData.Form.Init()
		}
	}

	form, cmd := m.priorityData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.priorityData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.priorityData.Form.State == huh.StateCompleted {
		m.mode = detailView
		if m.pendingIssue != nil {
			priority := m.priorityData.SelectedPriority
			cmds = append(cmds, m.postPriorityCmd(m.pendingIssue.Key, priority))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderEditPriorityView() string {
	var modalContent strings.Builder

	modalContent.WriteString(m.priorityData.Form.View())

	label := "Priority"
	if m.pendingIssue != nil {
		label += " " + m.pendingIssue.Key
	}

	return m.renderModal(label, modalContent.String(), 0.2, 0.3)
}
