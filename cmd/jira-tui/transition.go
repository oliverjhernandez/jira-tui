package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type TransitionFormData struct {
	SelectedIndex int
	Form          *huh.Form
}

func NewTransitionFormData(transitions []jira.Transition) *TransitionFormData {
	options := make([]huh.Option[int], len(transitions))
	for i, t := range transitions {
		options[i] = huh.NewOption(t.Name, i)
	}

	t := &TransitionFormData{
		SelectedIndex: 0,
	}
	t.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Select new status").
				Options(options...).
				Value(&t.SelectedIndex),
		),
	).WithWidth(50)

	return t
}

type CancelReasonFormData struct {
	Reason string
	Form   *huh.Form
}

func NewCancelReasonFormData() *CancelReasonFormData {
	c := &CancelReasonFormData{
		Reason: "",
	}
	c.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewText().
				Title("Cancellation Reason").
				Value(&c.Reason).
				Lines(10),
		),
	).WithWidth(60)

	return c
}

func isCancelTransition(t jira.Transition) bool {
	name := strings.ToLower(t.Name)
	return strings.Contains(name, "cancel") || strings.Contains(name, "cancelado")
}

func (m model) updateTransitionView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.mode = m.previousMode
			m.transitions = nil
			return m, nil
		}
	}

	if m.transitionData != nil {
		form, cmd := m.transitionData.Form.Update(msg)
		if f, ok := form.(*huh.Form); ok {
			m.transitionData.Form = f
			cmds = append(cmds, cmd)
		}

		if m.transitionData.Form.State == huh.StateCompleted {
			if len(m.transitions) > 0 {
				transition := m.transitions[m.transitionData.SelectedIndex]
				if m.activeIssue != nil && m.activeIssue.OriginalEstimate == "" {
					m.pendingTransition = &transition
					m.estimateData = NewEstimateFormData()
					m.mode = estimateView
					return m, m.estimateData.Form.Init()
				}
				if isCancelTransition(transition) {
					m.pendingTransition = &transition
					m.cancelReasonData = NewCancelReasonFormData()
					m.mode = cancelReasonView
					return m, m.cancelReasonData.Form.Init()
				}
				m.mode = detailView
				m.loadingCount++
				m.statusMessage.content = "Transitioning..."
				cmds = append(cmds, m.postTransitionCmd(m.activeIssue.Key, transition.ID, transition.Name))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderTransitionView() string {
	var bg *lipgloss.Layer
	if m.previousMode == detailView {
		bg = lipgloss.NewLayer(m.renderDetailView())
	} else if m.previousMode == listView {
		bg = lipgloss.NewLayer(m.renderListView())
	}

	var modalContent strings.Builder

	if m.transitions != nil {
		modalContent.WriteString(m.transitionData.Form.View())
	}

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.2)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.3)

	styledModal := ui.RenderPanelWithLabel("Transition "+m.activeIssue.Key, modalContent.String(), modalWidth, true)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}

func (m model) updatePostCancelReasonView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = m.previousMode
			m.pendingTransition = nil
			return m, m.cancelReasonData.Form.Init()
		}
	}

	form, cmd := m.cancelReasonData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.cancelReasonData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.cancelReasonData.Form.State == huh.StateCompleted {
		m.mode = detailView
		reason := m.cancelReasonData.Reason
		if m.pendingTransition != nil {
			transition := m.pendingTransition
			m.pendingTransition = nil
			m.loadingCount++
			m.statusMessage.content = "Transitioning " + m.activeIssue.Key
			cmds = append(cmds, m.postTransitionWithReasonCmd(m.activeIssue.Key, transition.ID, reason))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderPostCancelReasonView() string {
	var bg *lipgloss.Layer
	if m.mode == detailView {
		bg = lipgloss.NewLayer(m.renderDetailView())
	} else {
		bg = lipgloss.NewLayer(m.renderListView())
	}

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(ui.DimTextStyle.Render("Please provide a reason for canceling this issue:") + "\n\n")
	modalContent.WriteString(m.cancelReasonData.Form.View())

	styledModal := ui.ModalStyle.Render(modalContent.String())

	modalWidth := lipgloss.Width(styledModal)
	modalHeight := lipgloss.Height(styledModal)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
