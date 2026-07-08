package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type TransitionFormData struct {
	SelectedIndex int
	Transitions   []jira.Transition
	Form          *huh.Form
}

func NewTransitionFormData(transitions []jira.Transition) *TransitionFormData {
	options := make([]huh.Option[int], len(transitions))
	for i, t := range transitions {
		options[i] = huh.NewOption(t.Name, i)
	}

	t := &TransitionFormData{
		SelectedIndex: 0,
		Transitions:   transitions,
	}
	t.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
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

func isBlockedTransition(t jira.Transition) bool {
	return strings.Contains(strings.ToLower(t.Name), "bloq")
}

type BlockReasonFormData struct {
	Reason string
	Form   *huh.Form
}

func NewBlockReasonFormData() *BlockReasonFormData {
	b := &BlockReasonFormData{
		Reason: "",
	}
	b.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Motivo del Bloqueante").
				Value(&b.Reason),
		),
	).WithWidth(60)

	return b
}

func (m model) updateTransitionView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.mode = m.previousMode
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
			if m.pendingIssue != nil {
				idx := m.transitionData.SelectedIndex
				if idx < 0 || idx >= len(m.transitionData.Transitions) {
					m.mode = m.previousMode
					return m, tea.Batch(cmds...)
				}
				transition := m.transitionData.Transitions[idx]
				if m.pendingIssue.OriginalEstimate == "" {
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
				if isBlockedTransition(transition) {
					m.pendingTransition = &transition
					m.blockReasonData = NewBlockReasonFormData()
					m.mode = blockReasonView
					return m, m.blockReasonData.Form.Init()
				}
				m.mode = detailView
				m.loadingCount++
				m.statusMessage.content = "Transitioning..."
				cmds = append(cmds, m.postTransitionCmd(m.pendingIssue.Key, transition.ID, transition.Name))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderTransitionView() string {
	var modalContent strings.Builder

	if m.transitionData != nil {
		modalContent.WriteString(m.transitionData.Form.View())
	}

	label := "Transition"
	if m.pendingIssue != nil {
		label += " " + m.pendingIssue.Key
	}

	return m.renderModal(label, modalContent.String(), 0.2, 0.2)
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
		if m.pendingTransition != nil && m.pendingIssue != nil {
			transition := m.pendingTransition
			m.pendingTransition = nil
			m.loadingCount++
			m.statusMessage.content = "Transitioning " + m.pendingIssue.Key
			cmds = append(cmds, m.postTransitionWithReasonCmd(m.pendingIssue.Key, transition.ID, reason))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderPostCancelReasonView() string {
	var modalContent strings.Builder

	if m.pendingIssue != nil {
		header := ui.DetailHeaderStyle.Render(m.pendingIssue.Key) + " " + ui.RenderStatusBadge(m.pendingIssue.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(m.cancelReasonData.Form.View())

	return m.renderModal("Cancel Reason", modalContent.String(), 0.3, 0.2)
}

func (m model) updateBlockReasonView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = m.previousMode
			m.pendingTransition = nil
			return m, m.blockReasonData.Form.Init()
		}
	}

	form, cmd := m.blockReasonData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.blockReasonData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.blockReasonData.Form.State == huh.StateCompleted {
		m.mode = detailView
		reason := m.blockReasonData.Reason
		if m.pendingTransition != nil && m.pendingIssue != nil {
			transition := m.pendingTransition
			m.pendingTransition = nil
			m.loadingCount++
			m.statusMessage.content = "Transitioning " + m.pendingIssue.Key
			cmds = append(cmds, m.postBlockedTransitionCmd(m.pendingIssue.Key, transition.ID, reason))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderBlockReasonView() string {
	var modalContent strings.Builder

	if m.pendingIssue != nil {
		header := ui.DetailHeaderStyle.Render(m.pendingIssue.Key) + " " + ui.RenderStatusBadge(m.pendingIssue.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(m.blockReasonData.Form.View())

	return m.renderModal("Block Reason", modalContent.String(), 0.3, 0.2)
}
