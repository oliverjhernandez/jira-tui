package main

import (
	"fmt"
	"sort"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

// projectBoardJQL lists a project's top-level epics and tasks.
func projectBoardJQL(key string) string {
	return fmt.Sprintf(`project = %q AND issuetype in (Epic, Task) ORDER BY status DESC`, key)
}

type ProjectPickerFormData struct {
	SelectedIndex int
	Projects      []jira.Project
	Form          *huh.Form
}

func NewProjectPickerFormData(projects []jira.Project, visibleRows int) *ProjectPickerFormData {
	sorted := make([]jira.Project, len(projects))
	copy(sorted, projects)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	options := make([]huh.Option[int], len(sorted))
	for i, p := range sorted {
		options[i] = huh.NewOption(p.Name, i)
	}

	if visibleRows < 3 {
		visibleRows = 3
	}

	d := &ProjectPickerFormData{SelectedIndex: 0, Projects: sorted}
	d.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Open Project (type to filter)").
				Options(options...).
				// Bound the visible options so a long list scrolls inside the
				// modal instead of overflowing the screen.
				Height(visibleRows).
				Value(&d.SelectedIndex),
		),
	).WithWidth(50)

	return d
}

// projectPickerHScale is the modal height as a fraction of the terminal; the
// select's visible rows are sized to match so the list scrolls within it.
const projectPickerHScale = 0.6

func (m model) openProjectPicker() (tea.Model, tea.Cmd) {
	if len(m.projects) == 0 {
		m.setInfo("No projects loaded yet")
		return m, m.clearStatusAfter(clearMsgTimeout)
	}
	m.previousMode = m.mode
	visibleRows := int(float64(m.windowHeight)*projectPickerHScale) - 6 // title/help/borders
	m.projectPickerData = NewProjectPickerFormData(m.projects, visibleRows)
	m.mode = projectPickerView
	return m, m.projectPickerData.Form.Init()
}

func (m model) updateProjectPickerView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = m.previousMode
			m.projectPickerData = nil
			return m, nil
		}
	}

	form, cmd := m.projectPickerData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.projectPickerData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.projectPickerData.Form.State == huh.StateCompleted {
		idx := m.projectPickerData.SelectedIndex
		projects := m.projectPickerData.Projects
		m.projectPickerData = nil
		if idx >= 0 && idx < len(projects) {
			p := projects[idx]
			return m.openBoardTab(p.Name, projectBoardJQL(p.Key), tabProjectBoard)
		}
		m.mode = m.previousMode
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderProjectPickerView() string {
	var content string
	if m.projectPickerData != nil {
		content = m.projectPickerData.Form.View()
	}
	return m.renderModal("Open Project", content, 0.4, projectPickerHScale)
}
