package main

import (
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateTransitionView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
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
	return m, nil
}

func (m model) renderTransitionView() string {
	log.Printf("=== renderTransitionView called ===")

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Change Status for %s\n", m.selectedIssue.Key))
	b.WriteString(strings.Repeat("=", 50) + "\n\n")

	if m.loadingTransitions {
		b.WriteString("Loading available transitions...\n")
	} else if len(m.transitions) == 0 {
		b.WriteString("No transitions available for this issue.\n")
	} else {
		b.WriteString("Select new status:\n\n")
		for i, transition := range m.transitions {
			cursor := " "
			if m.transitionCursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, transition.Name))
		}
	}

	b.WriteString("\nPress j/k or ↑/↓ to navigate, Enter to select, Esc to cancel.\n")

	return b.String()
}
