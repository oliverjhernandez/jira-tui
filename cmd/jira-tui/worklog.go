package main

import (
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type WorklogFormData struct {
	Time string
	Date string
	Note string
	Form *huh.Form
}

func NewWorklogFormData() *WorklogFormData {
	w := &WorklogFormData{
		Time: "",
		Date: time.Now().Format("2006-01-02"),
		Note: "",
	}
	w.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Time").
				Placeholder("1h 30m").
				Value(&w.Time),
			huh.NewInput().
				Title("Date").
				Placeholder("2006-01-02").
				Value(&w.Date),
			huh.NewInput().
				Title("Note").
				Placeholder("Optional").
				Value(&w.Note),
		),
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(40)

	return w
}

func (m model) updatePostWorklogView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			m.postingComment = false
			return m, m.worklogData.Form.Init()
		}

	}

	form, cmd := m.worklogData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.worklogData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.worklogData.Form.State == huh.StateCompleted {
		m.mode = detailView
		m.postingWorkLog = false
		log.Printf("User ID: %s", m.myself.ID)
		timeSeconds, _ := parseTimeToSeconds(m.worklogData.Time)
		cmds = append(cmds, m.postWorkLog(m.selectedIssue.ID, m.worklogData.Date, m.myself.ID, timeSeconds))
	}
	return m, tea.Batch(cmds...)
}

func (m model) renderPostWorklogView() string {
	log.Printf("=== renderPostWorklogView called ===")

	bg := m.renderDetailView()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalWidth := m.getLargeModalWidth()
	modalHeight := m.getModalHeight(0.6)

	m.editTextArea.SetWidth(modalWidth - 6)
	m.editTextArea.SetHeight(modalHeight - 8)

	modalContent.WriteString(m.worklogData.Form.View())
	modalContent.WriteString("\n\n")
	footer := strings.Join([]string{
		ui.RenderKeyBind("tab", "next field"),
		ui.RenderKeyBind("enter", "submit"),
		ui.RenderKeyBind("esc", "cancel"),
	}, "  ")
	modalContent.WriteString(footer)

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
