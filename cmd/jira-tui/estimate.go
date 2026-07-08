package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

type EstimateFormData struct {
	Estimate string
	Form     *huh.Form
}

func NewEstimateFormData() *EstimateFormData {
	e := &EstimateFormData{
		Estimate: "",
	}
	e.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Original Estimate").
				Placeholder("1h 30m").
				Value(&e.Estimate),
		),
	).WithWidth(40)

	return e
}

func (m model) updatePostEstimateView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = detailView
			m.pendingTransition = nil
			return m, m.estimateData.Form.Init()
		}
	}

	form, cmd := m.estimateData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.estimateData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.estimateData.Form.State == huh.StateCompleted {
		m.mode = detailView
		if m.pendingIssue != nil {
			cmds = append(cmds, m.postEstimateCmd(m.pendingIssue.Key, m.estimateData.Estimate))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderPostEstimateView() string {
	return m.renderModal("Original Estimate", m.estimateData.Form.View(), 0.2, 0.1)
}
