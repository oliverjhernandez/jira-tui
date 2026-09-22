package main

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

const dateLayout = ui.DateLayout

type DatesFormData struct {
	StartDate string
	DueDate   string
	Form      *huh.Form
}

func validateDate(s string) error {
	if s == "" {
		return nil
	}
	if _, err := time.Parse(dateLayout, s); err != nil {
		return fmt.Errorf("use YYYY-MM-DD, or leave empty to clear")
	}
	return nil
}

func NewDatesFormData(startDate, dueDate string, startEditable bool) *DatesFormData {
	d := &DatesFormData{
		StartDate: startDate,
		DueDate:   dueDate,
	}

	fields := []huh.Field{}
	if startEditable {
		fields = append(fields, huh.NewInput().
			Title("Start date").
			Placeholder(dateLayout).
			Validate(validateDate).
			Value(&d.StartDate))
	}
	fields = append(fields, huh.NewInput().
		Title("Due date").
		Placeholder(dateLayout).
		Validate(validateDate).
		Value(&d.DueDate))

	d.Form = huh.NewForm(huh.NewGroup(fields...)).WithWidth(40)

	return d
}

func (m model) openDatesForm() (tea.Model, tea.Cmd) {
	if m.activeIssue == nil {
		return m, nil
	}

	m.setPendingIssue(m.activeIssue)
	m.datesData = NewDatesFormData(m.activeIssue.StartDate, m.activeIssue.DueDate, m.startDateFieldID != "")
	m.previousMode = m.mode
	m.mode = datesView

	var cmds []tea.Cmd
	cmds = append(cmds, m.datesData.Form.Init())
	if m.startDateFieldID == "" {
		m.setInfo("Start date field not found in Jira; editing due date only")
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
	}

	return m, tea.Batch(cmds...)
}

func (m model) updateDatesView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = m.previousMode
			m.datesData = nil
			return m, nil
		}
	}

	form, cmd := m.datesData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.datesData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.datesData.Form.State == huh.StateCompleted {
		start, due := m.datesData.StartDate, m.datesData.DueDate
		m.datesData = nil
		m.mode = m.previousMode

		if m.pendingIssue != nil {
			m.loadingCount++
			cmds = append(cmds, m.postDatesCmd(m.pendingIssue.Key, start, due))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderDatesView() string {
	var content string
	if m.datesData != nil {
		content = m.datesData.Form.View()
	}
	return m.renderModal("Dates", content, 0.25, 0.2)
}

var startDateFieldNames = []string{
	"start date",
	"fecha de inicio",
	"fecha inicio",
	"start",
}

func resolveStartDateField(fields []jira.Field) string {
	return resolveFieldByNames(fields, startDateFieldNames)
}

func resolveFieldByNames(fields []jira.Field, names []string) string {
	for _, want := range names {
		for _, f := range fields {
			if strings.EqualFold(strings.TrimSpace(f.Name), want) {
				return f.ID
			}
		}
	}
	return ""
}
