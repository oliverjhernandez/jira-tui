package main

import (
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type NewIssueFormData struct {
	ProjectName      string
	IssueTypeName    string
	ParentIssueKey   string
	ReporterName     string
	OriginalEstimate string
	Summary          string
	AssigneeName     string
	PriorityName     string
	DueDate          string
	Description      string
	Form             *huh.Form
}

func (m model) NewIssueForm() *NewIssueFormData {
	i := &NewIssueFormData{
		ProjectName:      "",
		IssueTypeName:    "",
		ParentIssueKey:   "",
		OriginalEstimate: "",
		Summary:          "",
		AssigneeName:     "",
		PriorityName:     "",
		DueDate:          "",
		Description:      "",
	}

	var commonProjects = []string{"DEV", "DCSDM", "ITELMEX", "EL"}

	var projectNames []huh.Option[string]
	for _, p := range commonProjects {
		projectNames = append(projectNames, huh.NewOption(p, p))
	}

	// var issueTypes []huh.Option[string]
	// for _, t := range m.issueTypes {
	// 	issueTypes = append(issueTypes, huh.NewOption(t.Name, t.Name))
	// }

	var commonTypeNames = []string{"Task", "Story", "Bug", "Epic"}

	var issueTypes []huh.Option[string]
	for _, t := range commonTypeNames {
		issueTypes = append(issueTypes, huh.NewOption(t, t))
	}

	var assigneeOptions []huh.Option[string]
	assigneeOptions = append(assigneeOptions,
		huh.NewOption("Me ("+m.myself.Name+")", m.myself.Name),
		huh.NewOption("Unassigned", ""),
	)

	// var users []huh.Option[string]
	// for _, u := range m.usersCache {
	// 	users = append(users, huh.NewOption(u.Name, u.Name))
	// }

	var priorities []huh.Option[string]
	for _, p := range m.priorities {
		priorities = append(priorities, huh.NewOption(p.Name, p.Name))
	}

	i.Form = huh.NewForm(

		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project").
				Options(projectNames...).
				Value(&i.ProjectName),
			huh.NewSelect[string]().
				Title("Issue Type").
				Options(issueTypes...).
				Value(&i.IssueTypeName),
			// huh.NewInput().
			// 	Title("ParentIssueKey").
			// 	Placeholder("DEV-123").
			// 	Value(&i.ParentIssueKey),
			// huh.NewSelect[string]().
			// 	Title("Reporter").
			// 	Options(users...).
			// 	Value(&i.ReporterName),
			huh.NewInput().
				Title("OriginalEstimate").
				Placeholder("1h").
				Value(&i.OriginalEstimate),
			huh.NewInput().
				Title("Summary").
				Placeholder("Summary").
				Value(&i.Summary),
			huh.NewSelect[string]().
				Title("Assignee").
				Options(assigneeOptions...).
				Value(&i.AssigneeName),
			huh.NewSelect[string]().
				Title("Priority").
				Options(priorities...).
				Value(&i.PriorityName),
			huh.NewInput().
				Title("DueDate").
				Placeholder(time.Now().Format("2006-01-02")).
				Value(&i.DueDate),

			huh.NewText().
				Title("Description").
				Placeholder("Improve something...").
				Value(&i.Description),
		),
	).WithWidth(40)

	return i
}

func (m model) updateNewIssueView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	form, cmd := m.newIssueData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.newIssueData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.newIssueData.Form.State == huh.StateCompleted {
		m.mode = listView
		cmds = append(cmds, m.postNewIssue(m.newIssueData))
	}

	if m.newIssueData.Form.State == huh.StateAborted {
		m.mode = detailView
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderNewIssueView() string {
	bg := m.renderSimpleBackground()

	var modalContent strings.Builder

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.7)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.6)

	m.textArea.SetWidth(modalWidth - 6)
	m.textArea.SetHeight(modalHeight - 8)

	modalContent.WriteString(m.newIssueData.Form.View())

	return ui.RenderCenteredModal(modalContent.String(), bg, m.windowWidth, m.windowHeight, ui.Modal3InputFormStyle)
}
