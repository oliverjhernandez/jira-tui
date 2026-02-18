package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(50)

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
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(60)

	return c
}

func isCancelTransition(t jira.Transition) bool {
	name := strings.ToLower(t.Name)
	return strings.Contains(name, "cancel") || strings.Contains(name, "cancelado")
}

func (m model) updateTransitionView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.mode = detailView
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
				if m.issueDetail != nil && m.issueDetail.OriginalEstimate == "" {
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
				cmds = append(cmds, m.postTransition(m.issueDetail.Key, transition.ID))
			}
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderTransitionView() string {
	bg := m.renderSimpleBackground()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := fmt.Sprintf("Change Status for %s", m.issueDetail.Key)
		modalContent.WriteString(header + "\n\n")
	}

	if m.loadingTransitions {
		modalContent.WriteString("Loading available transitions...\n")
	} else if len(m.transitions) == 0 {
		modalContent.WriteString("No transitions available for this issue.\n")
	} else {
		modalContent.WriteString(m.transitionData.Form.View())
	}

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowHeight, m.windowHeight, ui.ModalMultiSelectFormStyle)
}

func (m model) updatePostCancelReasonView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
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
			cmds = append(cmds, m.postTransitionWithReason(m.issueDetail.Key, transition.ID, reason))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderPostCancelReasonView() string {
	bg := m.renderDetailView()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(ui.StatusBarDescStyle.Render("Please provide a reason for canceling this issue:") + "\n\n")
	modalContent.WriteString(m.cancelReasonData.Form.View())

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowWidth, m.windowHeight, ui.ModalBlockInputStyle)
}
