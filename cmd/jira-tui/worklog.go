package main

import (
	"strconv"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
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
		if m.activeIssue == nil || m.myself == nil {
			m.statusMessage = statusMessage{
				msgType: errStatusBarMsg,
				content: "Cannot save worklog: missing issue or user",
			}
			return m, m.clearStatusAfter(clearMsgTimeout)
		}
		worklogID := strconv.Itoa(m.worklogFormData.ID)
		issueID := m.activeIssue.ID
		startDate := m.worklogFormData.StartDate
		accountID := m.myself.ID
		description := m.worklogFormData.Description
		time, _ := parseStringToSeconds(m.worklogFormData.Time)
		if m.editingWorklog {
			cmds = append(cmds, m.putWorkLogCmd(
				worklogID,
				issueID,
				startDate,
				accountID,
				description,
				time,
			))
			m.editingWorklog = false

		} else {
			cmds = append(cmds, m.postWorkLogCmd(
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
	return m.renderModal("Worklog", m.worklogFormData.Form.View(), 0.2, 0.2)
}
