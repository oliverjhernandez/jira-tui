package main

import (
	"fmt"
	"strings"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

// listGrouping selects how a tab's issues are grouped for display. It's a
// per-tab setting (see Tab.grouping) so a future keymap can toggle it on any
// tab; sectionsFor + buildListContent dispatch on the active tab's value.
type listGrouping int

const (
	groupStatus listGrouping = iota // group by status category (default)
	groupEpic                       // group tasks under their epic, in rank order
)

func groupingForKind(kind tabKind) listGrouping {
	if kind == tabProjectBoard {
		return groupEpic
	}
	return groupStatus
}

// currentGrouping is the active tab's grouping (defaults to groupStatus).
func (m model) currentGrouping() listGrouping {
	if m.activeTab >= 0 && m.activeTab < len(m.tabs) {
		return m.tabs[m.activeTab].grouping
	}
	return groupStatus
}

// sectionsFor builds the display sections for the active tab's grouping.
func (m *model) sectionsFor(issues []jira.Issue) []Section {
	if m.currentGrouping() == groupEpic {
		return groupByEpic(issues)
	}
	return m.classifyIssues(issues, m.statuses)
}

func isEpic(i jira.Issue) bool {
	return strings.Contains(strings.ToLower(i.Type), "epic")
}

// groupByEpic makes each epic a section (header) with its child tasks beneath,
// preserving the incoming (rank) order. Tasks whose epic isn't present go into a
// trailing "No epic" section.
func groupByEpic(issues []jira.Issue) []Section {
	order := make([]string, 0)
	byKey := make(map[string]*Section)

	for i := range issues {
		if isEpic(issues[i]) {
			key := issues[i].Key
			if _, ok := byKey[key]; !ok {
				epic := issues[i]
				byKey[key] = &Section{Name: epic.Summary, CategoryKey: key, Epic: &epic}
				order = append(order, key)
			}
		}
	}

	var noEpic []jira.Issue
	for i := range issues {
		if isEpic(issues[i]) {
			continue
		}
		parentKey := ""
		if issues[i].Parent != nil {
			parentKey = issues[i].Parent.Key
		}
		if s, ok := byKey[parentKey]; ok {
			s.Issues = append(s.Issues, issues[i])
		} else {
			noEpic = append(noEpic, issues[i])
		}
	}

	sections := make([]Section, 0, len(order)+1)
	for _, key := range order {
		sections = append(sections, *byKey[key])
	}
	if len(noEpic) > 0 {
		sections = append(sections, Section{Name: "No epic", CategoryKey: "no-epic", Issues: noEpic})
	}
	return sections
}

// buildEpicListContent renders the epic-grouped view: each epic as a header
// (key + summary + status), its tasks as normal rows. Rank order is preserved
// (no status sort).
func (m model) buildEpicListContent() string {
	var b strings.Builder

	sectionsToRender := m.sections
	if m.filteredSections != nil {
		sectionsToRender = m.filteredSections
	}

	for si, s := range sectionsToRender {
		if s.Epic != nil {
			title := ui.SectionTitleStyle.Render(fmt.Sprintf("%s  %s  (%d)", s.Epic.Key, s.Epic.Summary, len(s.Issues)))
			b.WriteString(title + "  " + ui.RenderStatusBadge(s.Epic.Status) + "\n")
		} else {
			b.WriteString(ui.SectionTitleStyle.Render(fmt.Sprintf("%s (%d)", s.Name, len(s.Issues))) + "\n")
		}

		for ii, issue := range s.Issues {
			selected := m.sectionCursor == si && m.cursor == ii
			dimmed := closureStatuses[issue.Status]
			b.WriteString(m.renderIssueRow(issue, selected, dimmed) + "\n")
		}
		b.WriteString("\n\n")
	}

	return b.String()
}
