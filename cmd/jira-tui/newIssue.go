package main

import (
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

type NewIssueFormData struct {
	ProjectName      string
	IssueTypeName    string
	ParentKey        string
	ReporterName     string
	OriginalEstimate string
	Summary          string
	AssigneeName     string
	PriorityName     string
	DueDate          string
	Description      string
	Form             *huh.Form
}

func (m model) NewIssueForm(issue *NewIssueFormData) *NewIssueFormData {
	var projectNames []huh.Option[string]

	// TODO: pass whole projects?, comparing names later seems flaky
	var projects []string
	if m.activeProjects != nil {
		for _, p := range m.activeProjects {
			projects = append(projects, p.Name)
		}
	}

	for _, p := range projects {
		projectNames = append(projectNames, huh.NewOption(p, p))
	}

	var commonTypeNames = []string{"Task", "Story", "Bug", "Epic"}

	var issueTypes []huh.Option[string]
	for _, t := range commonTypeNames {
		issueTypes = append(issueTypes, huh.NewOption(t, t))
	}

	var meName string
	if m.myself != nil {
		meName = m.myself.Name
	}
	var assigneeOptions []huh.Option[string]
	assigneeOptions = append(assigneeOptions,
		huh.NewOption("Me ("+meName+")", meName),
		huh.NewOption("Unassigned", ""),
	)

	var priorities []huh.Option[string]
	for _, p := range m.priorities {
		priorities = append(priorities, huh.NewOption(p.Name, p.Name))
	}

	issue.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Project").
				Options(projectNames...).
				Value(&issue.ProjectName),
			huh.NewSelect[string]().
				Title("Issue Type").
				Options(issueTypes...).
				Value(&issue.IssueTypeName),
			huh.NewInput().
				Title("ParentIssueKey").
				Placeholder("DEV-123").
				Value(&issue.ParentKey),
			// huh.NewSelect[string]().
			// 	Title("Reporter").
			// 	Options(users...).
			// 	Value(&i.ReporterName),
			huh.NewInput().
				Title("OriginalEstimate").
				Placeholder("1h").
				Value(&issue.OriginalEstimate),
			huh.NewInput().
				Title("Summary").
				Placeholder("Summary").
				Value(&issue.Summary),
			huh.NewSelect[string]().
				Title("Assignee").
				Options(assigneeOptions...).
				Value(&issue.AssigneeName),
			huh.NewSelect[string]().
				Title("Priority").
				Options(priorities...).
				Value(&issue.PriorityName),
			huh.NewInput().
				Title("DueDate").
				Placeholder(time.Now().Format("2006-01-02")).
				Value(&issue.DueDate),

			huh.NewText().
				Title("Description (Markdown supported)").
				Placeholder("Improve something...").
				Value(&issue.Description),
		),
	).WithWidth(40)

	return issue
}

func (m model) updateNewIssueView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = listView
			return m, m.newIssueData.Form.Init()
		}
	}

	form, cmd := m.newIssueData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.newIssueData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.newIssueData.Form.State == huh.StateCompleted {
		m.mode = listView
		m.loadingCount++
		m.statusMessage.content = "Posting new issue"
		cmds = append(cmds, m.postNewIssueCmd(m.newIssueData))
	}

	if m.newIssueData.Form.State == huh.StateAborted {
		m.mode = listView
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderNewIssueView() string {
	return m.renderModal("New Issue", m.newIssueData.Form.View(), 0.2, 0.6)
}
