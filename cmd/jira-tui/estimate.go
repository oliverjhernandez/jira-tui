package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
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
		cmds = append(cmds, m.postEstimate(m.activeIssue.Key, m.estimateData.Estimate))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderPostEstimateView() string {
	bg := lipgloss.NewLayer(m.renderDetailView())

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(m.estimateData.Form.View())

	styledModal := ui.ModalBlockInputStyle.Render(modalContent.String())

	modalWidth := lipgloss.Width(styledModal)
	modalHeight := lipgloss.Height(styledModal)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
