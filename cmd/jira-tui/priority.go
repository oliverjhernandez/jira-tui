package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
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
		priority := m.priorityData.SelectedPriority
		cmds = append(cmds, m.postPriorityCmd(m.issueDetail.Key, priority))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderEditPriorityView() string {
	var bg *lipgloss.Layer
	if m.previousMode == detailView {
		bg = lipgloss.NewLayer(m.renderDetailView())
	} else if m.previousMode == listView {
		bg = lipgloss.NewLayer(m.renderListView())
	}

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(m.priorityData.Form.View())
	modalWidth := ui.GetModalWidth(m.windowWidth, 0.2)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.3)

	styledModal := ui.RenderPanelWithLabel("Priority "+m.activeIssue.Key, modalContent.String(), modalWidth, true)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
