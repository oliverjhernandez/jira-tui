package main

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
			// Execute the selected transition
			if len(m.transitions) > 0 {
				transition := m.transitions[m.transitionCursor]
				return m, m.doTransition(m.selectedIssue.Key, transition.ID)
			}
		}
	}

	return m, nil
}

func (m model) renderTransitionView() string {
	log.Printf("=== renderTransitionView called ===")

	bg := m.renderDetailView()

	var modalContent strings.Builder
	modalContent.WriteString(fmt.Sprintf("Change Status for %s\n", m.selectedIssue.Key))
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
			modalContent.WriteString(fmt.Sprintf("%s %s\n", cursor, t.Name))
		}
	}

	modalContent.WriteString("\nPress j/k or ↑/↓ to navigate, Enter to select, Esc to cancel.\n")

	modalWidth := int(float64(m.windowWidth) * 0.7)
	modalHeight := int(float64(m.windowHeight) * 0.6)

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
