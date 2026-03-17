package main

import (
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type WorklogFormData struct {
	ID          int
	Time        string
	StartDate   string
	Description string
	Form        *huh.Form
}

func (m model) NewWorklogForm(w *jira.Worklog, width int) *WorklogFormData {
	wd := &WorklogFormData{
		ID:          w.ID,
		Time:        formatSecondsToString(w.Time),
		StartDate:   w.StartDate,
		Description: w.Description,
	}

	wd.Form =
		huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Time").
					Placeholder("1h 30m").
					Value(&wd.Time),
				huh.NewInput().
					Title("Date").
					Placeholder("2006-01-02").
					Value(&wd.StartDate),
				huh.NewInput().
					Title("Note").
					Placeholder("Optional").
					Value(&wd.Description),
			),
		).WithWidth(width)

	return wd
}

func (m model) updateWorklogView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = detailView
			return m, m.worklogFormData.Form.Init()
		}
	}

	form, cmd := m.worklogFormData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.worklogFormData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.worklogFormData.Form.State == huh.StateCompleted {
		m.mode = detailView
		worklogID := strconv.Itoa(m.worklogFormData.ID)
		issueID := m.issueDetail.ID
		startDate := m.worklogFormData.StartDate
		accountID := m.myself.ID
		description := m.worklogFormData.Description
		time, _ := parseStringToSeconds(m.worklogFormData.Time)
		if m.editingWorklog {
			cmds = append(cmds, m.putWorkLog(
				worklogID,
				issueID,
				startDate,
				accountID,
				description,
				time,
			))
			m.editingWorklog = false

		} else {
			cmds = append(cmds, m.postWorkLog(
				issueID,
				startDate,
				accountID,
				description,
				time,
			))
		}
	}
	return m, tea.Batch(cmds...)
}

func (m model) renderWorklogView() string {
	bg := lipgloss.NewLayer(m.renderDetailView())

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := ui.DetailHeaderStyle.Render(m.issueDetail.Key) + " " + ui.RenderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.7)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.6)

	m.textArea.SetWidth(modalWidth - 6)
	m.textArea.SetHeight(modalHeight - 8)

	modalContent.WriteString(m.worklogFormData.Form.View())

	styledModal := ui.ModalBlockInputStyle.Render(modalContent.String())

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
