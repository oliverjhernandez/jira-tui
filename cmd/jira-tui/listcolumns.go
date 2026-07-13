package main

import (
	"strings"

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
	header string
	width  func(c ui.ColumnWidths) int
	cell   func(m model, i jira.Issue, selected, dimmed bool) string
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
		cell:   func(m model, i jira.Issue, _, _ bool) string { return m.columnWidths.RenderDueDate(i.DueDate) },
	},
	{
		header: "LOGGED",
		width:  func(c ui.ColumnWidths) int { return c.TimeSpent },
		cell: func(m model, i jira.Issue, _, _ bool) string {
			return m.columnWidths.RenderTimeSpent(ui.FormatTimeSpent(m.worklogTotals[i.ID]))
		},
	},
}

// summaryCell renders the Summary column, including the parent-issue breadcrumb
// prefix and selected/dimmed styling.
func summaryCell(m model, i jira.Issue, selected, dimmed bool) string {
	var summaryText string
	if i.Parent != nil {
		parentPrefix := ui.IconEnter + " " + i.Parent.Key + " " + ui.IconSeparator + " "
		full := ui.TruncateLongString(parentPrefix+i.Summary, m.columnWidths.Summary)
		switch {
		case selected:
			summaryText = full
		case dimmed:
			summaryText = ui.DimTextStyle.Render(full)
		default:
			summaryText = ui.DimTextStyle.Render(parentPrefix) + strings.TrimPrefix(full, parentPrefix)
		}
	} else {
		summaryText = ui.TruncateLongString(i.Summary, m.columnWidths.Summary)
	}
	return m.columnWidths.RenderSummary(summaryText, selected, dimmed)
}

// rowPrefix is the 2-cell cursor gutter. Both states are exactly 2 cells wide so
// columns never shift horizontally between selected and unselected rows.
func rowPrefix(selected bool) string {
	if selected {
		return ui.IconCursor + " "
	}
	return "  "
}

// renderIssueRow builds one data row from the column model.
func (m model) renderIssueRow(i jira.Issue, selected, dimmed bool) string {
	cells := make([]string, len(listColumns))
	for ci, col := range listColumns {
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
	cells := make([]string, len(listColumns))
	for ci, col := range listColumns {
		cells[ci] = ui.PadCell(ui.ColumnHeaderStyle.Render(col.header), col.width(m.columnWidths))
	}
	header := "  " + strings.Join(cells, " ")
	rule := ui.ColumnHeaderRuleStyle.Render(strings.Repeat("─", lipgloss.Width(header)))
	return header + "\n" + rule
}
