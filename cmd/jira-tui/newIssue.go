package main

import (
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
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

	var assigneeOptions []huh.Option[string]
	assigneeOptions = append(assigneeOptions,
		huh.NewOption("Me ("+m.myself.Name+")", m.myself.Name),
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
				Title("Description").
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
	bg := lipgloss.NewLayer(m.renderListView())

	var modalContent strings.Builder

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.7)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.6)

	m.textArea.SetWidth(modalWidth - ui.PanelOverheadWidth)
	m.textArea.SetHeight(modalHeight - ui.PanelOverheadHeight)

	modalContent.WriteString(m.newIssueData.Form.View())

	styledModal := ui.ModalBlockInputStyle.Render(modalContent.String())

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
