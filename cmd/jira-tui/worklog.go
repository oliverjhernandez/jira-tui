package main

import (
	"log"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
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

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.7)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.6)

	m.editTextArea.SetWidth(modalWidth - 6)
	m.editTextArea.SetHeight(modalHeight - 8)

	modalContent.WriteString(m.worklogData.Form.View())

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowWidth, m.windowHeight, ui.Modal3InputFormStyle)
}
