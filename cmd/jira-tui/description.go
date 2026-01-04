package main

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func (m model) updateEditDescriptionView(msg tea.Msg) (tea.Model, tea.Cmd) {

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			m.editingDescription = false
			m.editTextArea.Blur()
			return m, nil
		case "alt+enter": // actually shift # NOTE: maybe i should deal with this
			m.mode = detailView
			m.editingDescription = false
			m.editTextArea.Blur()
			return m, m.updateDescription(m.issueDetail.Key, m.editTextArea.Value())
		}
	}

	var cmd tea.Cmd
	m.editTextArea, cmd = m.editTextArea.Update(msg)

	return m, cmd
}

func (m model) renderEditDescriptionView() string {
	log.Printf("=== renderEditDescriptionView called ===")

	background := m.renderDetailView()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + renderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalWidth := int(float64(m.windowWidth) * 0.7)
	modalHeight := int(float64(m.windowHeight) * 0.6)
	// modalY := (m.windowHeight-modalHeight)/2 - 5
	// modalX := (m.windowWidth / 2) - (modalWidth / 2)

	m.editTextArea.SetWidth(modalWidth - 6)
	m.editTextArea.SetHeight(modalHeight - 8)

	modalContent.WriteString("Description:\n")
	modalContent.WriteString(m.editTextArea.View() + "\n\n")
	modalContent.WriteString("shift+Enter save | esc cancel")

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight).
		Background(lipgloss.Color("235"))

	styledModal := modalStyle.Render(modalContent.String())

	canvas := background + styledModal
	return canvas
}
