package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func isCancelTransition(t jira.Transition) bool {
	name := strings.ToLower(t.Name)
	return strings.Contains(name, "cancel") || strings.Contains(name, "cancelado")
}

func (m model) updateTransitionView(msg tea.Msg) (tea.Model, tea.Cmd) {

	if keyMsg, ok := msg.(tea.KeyMsg); ok {

		switch keyMsg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc", "backspace":
			m.mode = detailView
			m.transitions = nil
		case "up", "k":
			if m.transitionCursor > 0 {
				m.transitionCursor--
			}
		case "down", "j":
			if m.transitionCursor < len(m.transitions)-1 {
				m.transitionCursor++
			}
		case "enter":
			if len(m.transitions) > 0 {
				transition := m.transitions[m.transitionCursor]
				if m.issueDetail != nil && m.issueDetail.OriginalEstimate == "" {
					m.pendingTransition = &transition
					m.estimateData = NewEstimateFormData()
					m.mode = postEstimateView
					return m, m.estimateData.Form.Init()
				}
				if isCancelTransition(transition) {
					m.pendingTransition = &transition
					m.editTextArea.Reset()
					m.editTextArea.Focus()
					m.mode = postCancelReasonView
					return m, textarea.Blink
				}
				return m, m.postTransition(m.selectedIssue.Key, transition.ID)
			}
		}
	}

	return m, nil
}

func (m model) renderTransitionView() string {
	log.Printf("=== renderTransitionView called ===")

	bg := m.renderDetailView()

	var modalContent strings.Builder
	fmt.Fprintf(&modalContent, "Change Status for %s\n", m.selectedIssue.Key)
	modalContent.WriteString(strings.Repeat("=", 50) + "\n\n")

	if m.loadingTransitions {
		modalContent.WriteString("Loading available transitions...\n")
	} else if len(m.transitions) == 0 {
		modalContent.WriteString("No transitions available for this issue.\n")
	} else {
		modalContent.WriteString("Select new status:\n\n")
		for i, t := range m.transitions {
			cursor := " "
			if m.transitionCursor == i {
				cursor = ">"
			}
			fmt.Fprintf(&modalContent, "%s %s\n", cursor, t.Name)
		}
	}

	footer := strings.Join([]string{
		ui.RenderKeyBind("j/k", "navigate"),
		ui.RenderKeyBind("enter", "select"),
		ui.RenderKeyBind("esc", "cancel"),
	}, "  ")
	modalContent.WriteString("\n" + footer + "\n")

	modalWidth := m.getSmallModalWidth()
	modalHeight := m.getModalHeight(0.4)

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight).
		Background(lipgloss.Color("235"))

	styledModal := modalStyle.Render(modalContent.String())
	overlay := PlaceOverlay(10, 20, styledModal, bg, false)

	return overlay
}

func (m model) updatePostCancelReasonView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			m.pendingTransition = nil
			m.editTextArea.Reset()
			m.editTextArea.Blur()
			return m, nil
		case "alt+enter":
			reason := m.editTextArea.Value()
			m.editTextArea.Reset()
			m.editTextArea.Blur()
			if m.pendingTransition != nil {
				transition := m.pendingTransition
				m.pendingTransition = nil
				return m, m.postTransitionWithReason(m.selectedIssue.Key, transition.ID, reason)
			}
			m.mode = detailView
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.editTextArea, cmd = m.editTextArea.Update(msg)

	return m, cmd
}

func (m model) renderPostCancelReasonView() string {
	log.Printf("=== renderPostCancelReasonView called ===")

	bg := m.renderDetailView()

	var modalContent strings.Builder

	modalContent.WriteString(ui.SectionTitleStyle.Render("Cancellation Reason") + "\n\n")

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(ui.StatusBarDescStyle.Render("Please provide a reason for canceling this issue:") + "\n\n")

	modalWidth := m.getMediumModalWidth()
	modalHeight := m.getModalHeight(0.5)

	m.editTextArea.SetWidth(modalWidth - 6)
	m.editTextArea.SetHeight(modalHeight - 12)

	modalContent.WriteString(m.editTextArea.View() + "\n\n")
	cancelFooter := strings.Join([]string{
		ui.RenderKeyBind("shift+enter", "submit"),
		ui.RenderKeyBind("esc", "cancel"),
	}, "  ")
	modalContent.WriteString(cancelFooter)

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight).
		Background(lipgloss.Color("235"))

	styledModal := modalStyle.Render(modalContent.String())
	overlay := PlaceOverlay(10, 20, styledModal, bg, false)

	return overlay
}
