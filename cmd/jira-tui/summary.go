package main

import (
	"errors"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

type SummaryFormData struct {
	Summary string
	Form    *huh.Form
}

func NewSummaryFormData(initialValue string) *SummaryFormData {
	s := &SummaryFormData{
		Summary: initialValue,
	}
	s.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Summary").
				CharLimit(255). // Jira's summary field maximum
				Value(&s.Summary).
				Validate(func(v string) error {
					if strings.TrimSpace(v) == "" {
						return errors.New("summary cannot be empty")
					}
					return nil
				}),
		),
	).WithWidth(60)

	return s
}

func (m model) updateEditSummaryView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = detailView
			return m, m.summaryData.Form.Init()
		}
	}

	form, cmd := m.summaryData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.summaryData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.summaryData.Form.State == huh.StateCompleted {
		m.mode = detailView
		summary := strings.TrimSpace(m.summaryData.Summary)
		m.loadingCount++
		cmds = append(cmds, m.updateSummaryCmd(m.activeIssue.Key, summary))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderEditSummaryView() string {
	return m.renderModal("Summary", m.summaryData.Form.View(), 0.3, 0.2)
}
