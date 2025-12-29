package main

import (
	"log"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

func (m model) updateEditPriorityView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			m.editingPriority = false
			return m, nil
		case "up", "k":
			log.Printf("p cursor: %d", m.priorityCursor)
			if m.priorityCursor > 0 {
				m.priorityCursor--
			}
			return m, nil
		case "down", "j":
			if m.priorityCursor < len(m.priorityOptions)-1 {
				m.priorityCursor++
			}
			return m, nil
		case "enter":
			m.mode = detailView
			m.editingPriority = false
			return m, m.postPriority(m.issueDetail.Key, m.priorityOptions[m.priorityCursor].Name)
		}
	}

	return m, nil
}

func (m model) renderEditPriorityView() string {
	background := m.renderDetailView()

	var modalContent strings.Builder

	if m.priorityOptions != nil {
		header := detailHeaderStyle.Render(m.issueDetail.Key) + " " + renderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalWidth := int(float64(m.windowWidth) * 0.3)
	modalHeight := int(float64(m.windowHeight) * 0.2)
	modalY := (m.windowHeight-modalHeight)/2 - 5
	modalX := (m.windowWidth / 2) - (modalWidth / 2)

	modalContent.WriteString("Priority:\n")

	for i, v := range m.priorityOptions {
		line := v.Name
		if m.priorityCursor == i {
			line = "> " + line
		} else {
			line = " " + line
		}

		modalContent.WriteString(line + "\n")
	}
	modalContent.WriteString("enter save | esc cancel")

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(modalWidth).
		Height(modalHeight).
		Background(lipgloss.Color("235"))

	styledModal := modalStyle.Render(modalContent.String())

	backgroundLayer := lipgloss.NewLayer(background)
	modalLayer := lipgloss.NewLayer(styledModal).
		Y(modalY).X(modalX)

	canvas := lipgloss.NewCanvas(backgroundLayer, modalLayer)
	return canvas.Render()
}
