package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
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

		if m.pendingIssue != nil {
			relation := m.issueLinkData.Relation
			toIssueKey := m.issueLinkData.IssueKey

			cmds = append(cmds, m.postLinkIssueCmd(
				m.pendingIssue.Key,
				toIssueKey,
				relation,
			))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderIssueLinkView() string {
	label := "Link"
	if m.pendingIssue != nil {
		label += " " + m.pendingIssue.Key
	}

	return m.renderModal(label, m.issueLinkData.Form.View(), 0.2, 0.3)
}
