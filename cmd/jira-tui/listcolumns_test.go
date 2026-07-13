package main

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

// TestListHeaderAlignsWithRows is the acceptance test for the feature: the
// header label row, the separator rule, and every data row (selected or not)
// must all be exactly the same display width, so labels sit над their data.
func TestListHeaderAlignsWithRows(t *testing.T) {
	issue := jira.Issue{
		Key:      "PROJ-123",
		Type:     "Bug",
		Status:   "In Progress",
		Summary:  "Fix the login crash on submit",
		Reporter: jira.Reporter{DisplayName: "Ana Ruiz"},
		Assignee: "Beto Sol",
		Priority: jira.Priority{Name: "High"},
		Created:  "2026-07-01",
		DueDate:  "2026-07-10",
	}
	unassigned := issue
	unassigned.Assignee = "Unassigned"

	for _, tw := range []int{200, 90} {
		cw := ui.CalculateColumnWidths(tw)
		m := model{columnWidths: cw}
		want := cw.TotalWidth()

		header := m.renderListColumnsHeader()
		lines := strings.Split(header, "\n")
		if len(lines) != 2 {
			t.Fatalf("tw=%d: header should be 2 lines (labels + rule), got %d", tw, len(lines))
		}
		for i, ln := range lines {
			if got := lipgloss.Width(ln); got != want {
				t.Errorf("tw=%d: header line %d width = %d, want %d", tw, i, got, want)
			}
		}

		cases := map[string]string{
			"selected":   m.renderIssueRow(issue, true, false),
			"unselected": m.renderIssueRow(issue, false, false),
			"dimmed":     m.renderIssueRow(issue, false, true),
			"unassigned": m.renderIssueRow(unassigned, false, false),
		}
		for name, row := range cases {
			if got := lipgloss.Width(row); got != want {
				t.Errorf("tw=%d: %s row width = %d, want %d", tw, name, got, want)
			}
		}
	}
}

// TestListColumnHeadersFit guards that every short label actually fits inside
// its column (a label wider than its cell would push the whole row out of
// alignment — the "human can't connect the name to the data" failure).
func TestListColumnHeadersFit(t *testing.T) {
	cw := ui.CalculateColumnWidths(200)
	for _, col := range listColumns {
		w := col.width(cw)
		if lipgloss.Width(col.header) > w {
			t.Errorf("header %q (width %d) does not fit its column width %d", col.header, lipgloss.Width(col.header), w)
		}
	}
}
