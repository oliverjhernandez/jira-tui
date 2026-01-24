package main

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
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
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(40)

	return e
}

func (m model) updatePostEstimateView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
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
		log.Printf("Estimate form completed with value: %s", m.estimateData.Estimate)
		cmds = append(cmds, m.postEstimate(m.selectedIssue.Key, m.estimateData.Estimate))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderPostEstimateView() string {
	log.Printf("=== renderPostEstimateView called ===")

	bg := m.renderDetailView()

	var modalContent strings.Builder

	modalContent.WriteString(ui.SectionTitleStyle.Render("Original Estimate Required") + "\n\n")

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalContent.WriteString(ui.StatusBarDescStyle.Render("You must set an original estimate before transitioning this issue.") + "\n\n")

	modalWidth := m.getModalWidth(0.5)
	modalHeight := m.getModalHeight(0.4)

	modalContent.WriteString(m.estimateData.Form.View())

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
