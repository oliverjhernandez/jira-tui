package main

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type IssueLinkFormData struct {
	IssueKey string
	Relation jira.LinkType
	Form     *huh.Form
}

func NewIssueLinkForm(width int) *IssueLinkFormData {
	linkTypes := []huh.Option[jira.LinkType]{
		huh.NewOption("Relates", jira.Relates),
		huh.NewOption("Blocks", jira.Blocks),
		huh.NewOption("Duplicates", jira.Duplicates),
	}
	ld := &IssueLinkFormData{}

	ld.Form =
		huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Key").
					Placeholder("DEV-123").
					Value(&ld.IssueKey),
				huh.NewSelect[jira.LinkType]().
					Title("Relation").
					Options(linkTypes...).
					Value(&ld.Relation),
			),
		).WithWidth(width)

	return ld
}

func (m model) updateIssueLinkView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = m.previousMode
			return m, nil
		}
	}

	form, cmd := m.issueLinkData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.issueLinkData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.issueLinkData.Form.State == huh.StateCompleted {
		m.mode = detailView

		relation := m.issueLinkData.Relation
		toIssueKey := m.issueLinkData.IssueKey

		cmds = append(cmds, m.postLinkIssueCmd(
			m.issueDetail.Key,
			toIssueKey,
			relation,
		))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderIssueLinkView() string {
	bg := lipgloss.NewLayer(m.renderDetailView())

	var modalContent strings.Builder

	modalWidth := ui.GetModalWidth(m.windowWidth, 0.2)
	modalHeight := ui.GetModalHeight(m.windowHeight, 0.3)

	m.textArea.SetWidth(modalWidth - ui.PanelOverheadWidth)
	m.textArea.SetHeight(modalHeight - ui.PanelOverheadHeight)

	modalContent.WriteString(m.issueLinkData.Form.View())

	styledModal := ui.RenderPanelWithLabel("Link "+m.activeIssue.Key, modalContent.String(), modalWidth, true)

	y := (m.windowHeight - modalHeight) / 2
	x := (m.windowWidth - modalWidth) / 2

	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	comp := lipgloss.NewCompositor(bg, fg)

	return comp.Render()
}
