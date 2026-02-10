package main

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
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
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(40)

	return e
}

func (m model) updatePostEstimateView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
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
		cmds = append(cmds, m.postEstimate(m.issueDetail.Key, m.estimateData.Estimate))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderPostEstimateView() string {
	bg := m.renderDetailView()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(m.estimateData.Form.View())

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowWidth, m.windowHeight, ui.ModalMultiSelectFormStyle)
}
