package main

import (
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

// listHeaderHeight is the number of lines the pinned column header occupies
// inside the list panel (the label row plus the separator rule).
const listHeaderHeight = 2

// listColumn describes one column of the issue list. The same descriptor drives
// both the pinned header and every data row, so labels always line up над the
// data they name — change the order/width here and both move together.
type listColumn struct {
	header  string
	width   func(c ui.ColumnWidths) int
	cell    func(m model, i jira.Issue, selected, dimmed bool) string
	enabled func(m model) bool
}

// layoutWidths reclaims the space of any hidden column for the summary, so the
// row still spans the same total width.
func (m model) layoutWidths(cw ui.ColumnWidths) ui.ColumnWidths {
	if !m.tempoEnabled {
		cw.Summary += cw.TimeSpent + cw.Empty
		cw.TimeSpent = 0
	}
	return cw
}

func (m model) visibleColumns() []listColumn {
	cols := make([]listColumn, 0, len(listColumns))
	for _, col := range listColumns {
		if col.enabled != nil && !col.enabled(m) {
			continue
		}
		cols = append(cols, col)
	}
	return cols
}

// listColumns is the ordered, single source of truth for the list layout.
var listColumns = []listColumn{
	{
		header: "TYPE",
		width:  func(c ui.ColumnWidths) int { return c.Type },
		cell:   func(m model, i jira.Issue, _, _ bool) string { return ui.RenderIssueType(i.Type, false) },
	},
	{
		header: "KEY",
		width:  func(c ui.ColumnWidths) int { return c.Key },
		cell:   func(m model, i jira.Issue, _, _ bool) string { return m.columnWidths.RenderKey(i.Key) },
	},
	{
		header: "STATUS",
		width:  func(c ui.ColumnWidths) int { return c.Status },
		cell:   func(m model, i jira.Issue, _, _ bool) string { return ui.RenderStatusBadge(i.Status) },
	},
	{
		header: "PRI",
		width:  func(c ui.ColumnWidths) int { return c.Priority },
		cell:   func(m model, i jira.Issue, _, _ bool) string { return ui.RenderPriority(i.Priority.Name, false) },
	},
	{
		header: "SUMMARY",
		width:  func(c ui.ColumnWidths) int { return c.Summary },
		cell:   summaryCell,
	},
	{
		header: "REPORTER",
		width:  func(c ui.ColumnWidths) int { return c.Reporter },
		cell: func(m model, i jira.Issue, _, _ bool) string {
			return m.columnWidths.RenderReporter("@" + i.Reporter.DisplayName)
		},
	},
	{
		header: "ASSIGNEE",
		width:  func(c ui.ColumnWidths) int { return c.Assignee },
		cell: func(m model, i jira.Issue, _, _ bool) string {
			a := i.Assignee
			if a != "" && a != "Unassigned" {
				a = "@" + a
			}
			if m.isMine(i) {
				return m.columnWidths.RenderAssigneeMine(a)
			}
			return m.columnWidths.RenderAssignee(a)
		},
	},
	{
		header: "CREATED",
		width:  func(c ui.ColumnWidths) int { return c.CreatedDate },
		cell:   func(m model, i jira.Issue, _, _ bool) string { return m.columnWidths.RenderCreatedDate(i.Created) },
	},
	{
		header: "DUE",
		width:  func(c ui.ColumnWidths) int { return c.DueDate },
		cell: func(m model, i jira.Issue, _, _ bool) string {
			return m.columnWidths.RenderDueDate(i.DueDate, time.Now(), isClosedIssue(i))
		},
	},
	{
		header: "LOGGED",
		width:  func(c ui.ColumnWidths) int { return c.TimeSpent },
		cell: func(m model, i jira.Issue, _, _ bool) string {
			return m.columnWidths.RenderTimeSpent(ui.FormatTimeSpent(m.worklogTotals[i.ID]))
		},
		enabled: func(m model) bool { return m.tempoEnabled },
	},
}

// summaryCell renders the Summary column, including the parent-issue breadcrumb
// prefix and selected/dimmed styling.
func summaryCell(m model, i jira.Issue, selected, dimmed bool) string {
	budget := m.columnWidths.Summary

	chips := ui.RenderTags(m.tagsOf(i.Key), tagCellWidth(budget), dimmed)
	if chipWidth := lipgloss.Width(chips); chipWidth > 0 {
		budget -= chipWidth + 1
	}

	var summaryText string
	if i.Parent != nil {
		parentPrefix := ui.IconEnter + " " + i.Parent.Key + " " + ui.IconSeparator + " "
		full := ui.TruncateLongString(parentPrefix+i.Summary, budget)
		switch {
		case selected:
			summaryText = full
		case dimmed:
			summaryText = ui.DimTextStyle.Render(full)
		default:
			summaryText = ui.DimTextStyle.Render(parentPrefix) + strings.TrimPrefix(full, parentPrefix)
		}
	} else {
		summaryText = ui.TruncateLongString(i.Summary, budget)
	}

	if chips != "" {
		summaryText = ui.PadCell(summaryText, budget) + " " + chips
	}
	return m.columnWidths.RenderSummary(summaryText, selected, dimmed)
}

const (
	// minSummaryTextWidth is how much of the SUMMARY cell is always left for
	// the summary itself. Tag chips may borrow the rest: SUMMARY sits at its
	// 50-column floor on any terminal under ~170, so chips that waited for
	// slack never appeared at all.
	minSummaryTextWidth = 36
	minTagCellWidth     = 8
	maxTagCellWidth     = 24
)

// tagCellWidth is how much of the SUMMARY cell tag chips may take. Only tagged
// rows pay: an issue with no tags keeps the whole cell.
func tagCellWidth(summaryWidth int) int {
	room := summaryWidth - minSummaryTextWidth - 1 // the separating space
	if room < minTagCellWidth {
		return 0
	}
	return min(room, maxTagCellWidth)
}

// rowPrefix is the 2-cell cursor gutter. Both states are exactly 2 cells wide so
// columns never shift horizontally between selected and unselected rows.
func rowPrefix(selected bool) string {
	if selected {
		return ui.IconCursor + " "
	}
	return "  "
}

// isMine reports whether the issue is assigned to the current user, matched by
// account id (a stable identifier) rather than display name.
func (m model) isMine(i jira.Issue) bool {
	return m.myself != nil && i.AssigneeID != "" && i.AssigneeID == m.myself.ID
}

// renderIssueRow builds one data row from the column model.
func (m model) renderIssueRow(i jira.Issue, selected, dimmed bool) string {
	cols := m.visibleColumns()
	cells := make([]string, len(cols))
	for ci, col := range cols {
		cells[ci] = ui.PadCell(col.cell(m, i, selected, dimmed), col.width(m.columnWidths))
	}
	line := strings.Join(cells, " ")
	if selected {
		return rowPrefix(true) + ui.SelectedRowStyle.Render(line)
	}
	return rowPrefix(false) + ui.NormalRowStyle.Render(line)
}

// renderListColumnsHeader builds the pinned header: the labels aligned to the
// same widths as the rows, plus a separator rule spanning the full row width.
func (m model) renderListColumnsHeader() string {
	cols := m.visibleColumns()
	cells := make([]string, len(cols))
	for ci, col := range cols {
		cells[ci] = ui.PadCell(ui.ColumnHeaderStyle.Render(col.header), col.width(m.columnWidths))
	}
	header := "  " + strings.Join(cells, " ")
	rule := ui.ColumnHeaderRuleStyle.Render(strings.Repeat("─", lipgloss.Width(header)))
	return header + "\n" + rule
}
