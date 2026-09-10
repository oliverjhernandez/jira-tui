package main

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

const longSummary = "Refactor the transition cache so board views stop refetching every status on each keypress"

func metadataPanelModel(summary string) model {
	m := newTabModel([]Tab{{id: 0, baseView: detailView}}, 0)
	m.mode = detailView
	m.activeIssue = &jira.Issue{
		Key:      "DEV-1234",
		Type:     "Task",
		Summary:  summary,
		Status:   "In Progress",
		Assignee: "Oliver Hernandez",
	}
	return m
}

func TestSummaryGrowsWithThePanel(t *testing.T) {
	t.Parallel()

	m := metadataPanelModel(longSummary)

	narrow := m.renderMetadataPanel(80, 7)
	wide := m.renderMetadataPanel(160, 7)

	narrowKept := visibleSummaryRunes(t, narrow, longSummary)
	wideKept := visibleSummaryRunes(t, wide, longSummary)

	if wideKept <= narrowKept {
		t.Errorf("summary did not grow with the panel: 80 cols kept %d runes, 160 cols kept %d", narrowKept, wideKept)
	}
	if wideKept != len([]rune(longSummary)) {
		t.Errorf("a 160-col panel has room for the whole summary, kept %d of %d runes:\n%s",
			wideKept, len([]rune(longSummary)), wide)
	}
}

func TestSummaryNoLongerCapsAtFifty(t *testing.T) {
	t.Parallel()

	m := metadataPanelModel(longSummary)
	panel := m.renderMetadataPanel(160, 7)

	if kept := visibleSummaryRunes(t, panel, longSummary); kept <= 50 {
		t.Errorf("summary still capped at the old fixed width: kept %d runes\n%s", kept, panel)
	}
}

func TestShortSummaryIsNotPadded(t *testing.T) {
	t.Parallel()

	m := metadataPanelModel("Short one")
	panel := m.renderMetadataPanel(160, 7)

	if !strings.Contains(panel, "Short one") {
		t.Errorf("short summary missing:\n%s", panel)
	}
	if strings.Contains(panel, "…") {
		t.Errorf("short summary should not be truncated:\n%s", panel)
	}
}

// Panels narrower than 80 already overflow on the fixed-width metadata columns,
// independently of the summary, so the header is checked from 80 up.
func TestHeaderNeverWrapsThePanel(t *testing.T) {
	t.Parallel()

	summaries := []string{
		"Short one",
		longSummary,
		strings.Repeat("verylongunbrokentoken", 30),
	}

	for _, width := range []int{80, 100, 120, 160, 240} {
		for _, summary := range summaries {
			m := metadataPanelModel(summary)
			m.activeIssue.Parent = &jira.Parent{Key: "DEV-1", Type: "Epic"}
			m.sections = []Section{{Name: "Doing", Issues: make([]jira.Issue, 25)}}
			m.sectionCursor = 0
			m.cursor = 3

			panel := m.renderMetadataPanel(width, 7)

			if got := lipgloss.Width(panel); got != width {
				t.Errorf("width=%d: panel rendered %d wide", width, got)
			}
			if got := len(strings.Split(panel, "\n")); got > 8 {
				t.Errorf("width=%d: header wrapped, panel grew to %d lines:\n%s", width, got, panel)
			}
		}
	}
}

// visibleSummaryRunes counts how many leading runes of summary survived into the
// rendered panel, ignoring the ellipsis the truncation appends.
func visibleSummaryRunes(t *testing.T, panel, summary string) int {
	t.Helper()

	plain := stripANSI(panel)
	runes := []rune(summary)
	for n := len(runes); n > 0; n-- {
		if strings.Contains(plain, string(runes[:n])) {
			return n
		}
	}
	return 0
}

func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape && (r == 'm' || r == '\\'):
			inEscape = false
		case !inEscape:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestSummaryTruncatesRatherThanOverflowing(t *testing.T) {
	t.Parallel()

	huge := strings.Repeat("x", 400)
	m := metadataPanelModel(huge)

	for _, width := range []int{80, 100, 160} {
		panel := m.renderMetadataPanel(width, 7)
		plain := stripANSI(panel)

		if !strings.Contains(plain, "…") {
			t.Errorf("width=%d: an over-long summary should be truncated:\n%s", width, panel)
		}
		if got := len(strings.Split(panel, "\n")); got > 8 {
			t.Errorf("width=%d: over-long summary wrapped the panel to %d lines", width, got)
		}
		if kept := strings.Count(plain, "x"); kept >= width-ui.PanelOverheadWidth {
			t.Errorf("width=%d: kept %d summary runes, more than the %d-col content area",
				width, kept, width-ui.PanelOverheadWidth)
		}
	}
}

func TestFormatEstimate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   string
		want string
	}{
		{"hours and minutes", "9000", "2h 30m"},
		{"whole hours", "10800", "3h"},
		{"minutes only", "1800", "30m"},
		{"zero is no estimate", "0", ""},
		{"absent field", "", ""},
		{"not a number", "3h", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatEstimate(tt.in); got != tt.want {
				t.Errorf("formatEstimate(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestJoinNonEmpty(t *testing.T) {
	t.Parallel()

	if got := joinNonEmpty("  ", "a", "", "b"); got != "a  b" {
		t.Errorf("joinNonEmpty dropped the wrong parts: %q", got)
	}
	if got := joinNonEmpty("  ", "", ""); got != "" {
		t.Errorf("all-empty should join to empty, got %q", got)
	}
	if got := joinNonEmpty("  ", "only"); got != "only" {
		t.Errorf("single part should not gain a separator: %q", got)
	}
}

func TestMetadataPanelShowsOriginalEstimate(t *testing.T) {
	t.Parallel()

	m := metadataPanelModel("Fix the thing")
	m.activeIssue.OriginalEstimate = "9000"

	panel := m.renderMetadataPanel(120, 7)
	plain := stripANSI(panel)

	if !strings.Contains(plain, "Est: 2h 30m") {
		t.Errorf("original estimate missing from the panel:\n%s", panel)
	}
	if got := len(strings.Split(panel, "\n")); got > 8 {
		t.Errorf("estimate added a line: panel is %d lines\n%s", got, panel)
	}
}

func TestMetadataPanelOmitsAbsentEstimate(t *testing.T) {
	t.Parallel()

	m := metadataPanelModel("Fix the thing")
	m.activeIssue.OriginalEstimate = ""

	plain := stripANSI(m.renderMetadataPanel(120, 7))

	if strings.Contains(plain, "Est:") {
		t.Errorf("issue with no estimate should not show the label:\n%s", plain)
	}
}

func TestEstimateDoesNotWrapTheHeader(t *testing.T) {
	t.Parallel()

	for _, width := range []int{80, 100, 120, 160, 240} {
		m := metadataPanelModel(longSummary)
		m.activeIssue.OriginalEstimate = "144000"
		m.activeIssue.Worklogs = []jira.Worklog{{Time: 45296}}

		panel := m.renderMetadataPanel(width, 7)

		if got := lipgloss.Width(panel); got != width {
			t.Errorf("width=%d: panel rendered %d wide", width, got)
		}
		if got := len(strings.Split(panel, "\n")); got > 8 {
			t.Errorf("width=%d: estimate + logged wrapped the panel to %d lines:\n%s", width, got, panel)
		}
	}
}

func TestIsClosedStatus(t *testing.T) {
	t.Parallel()

	closed := []string{"Done", "Cancelada"}
	open := []string{"Trabajando", "To Do", "Backlog", "Validación", "Ready to Deploy", "Selected for Development"}

	for _, s := range closed {
		if !isClosedStatus(s) {
			t.Errorf("isClosedStatus(%q) = false, want true", s)
		}
	}
	for _, s := range open {
		if isClosedStatus(s) {
			t.Errorf("isClosedStatus(%q) = true, want false", s)
		}
	}
}

func TestDetailPanelDoesNotAlarmOnClosedIssues(t *testing.T) {
	t.Parallel()

	overdue := "2026-01-15"

	openModel := metadataPanelModel("Still going")
	openModel.activeIssue.Status = "Trabajando"
	openModel.activeIssue.DueDate = overdue

	doneModel := metadataPanelModel("All finished")
	doneModel.activeIssue.Status = "Done"
	doneModel.activeIssue.DueDate = overdue

	cancelledModel := metadataPanelModel("Dropped")
	cancelledModel.activeIssue.Status = "Cancelada"
	cancelledModel.activeIssue.DueDate = overdue

	openPanel := openModel.renderMetadataPanel(120, 7)
	if !strings.Contains(openPanel, ui.IconDueOverdue) {
		t.Errorf("an open overdue issue should still alarm:\n%s", openPanel)
	}

	for _, tc := range []struct {
		name  string
		panel string
	}{
		{"Done", doneModel.renderMetadataPanel(120, 7)},
		{"Cancelada", cancelledModel.renderMetadataPanel(120, 7)},
	} {
		if strings.Contains(tc.panel, ui.IconDueOverdue) {
			t.Errorf("%s issue still shows the overdue alarm:\n%s", tc.name, tc.panel)
		}
		if !strings.Contains(tc.panel, ui.IconDueClosed) {
			t.Errorf("%s issue should show the settled-date icon:\n%s", tc.name, tc.panel)
		}
		if !strings.Contains(stripANSI(tc.panel), "Jan 15") {
			t.Errorf("%s issue should still show the date:\n%s", tc.name, tc.panel)
		}
	}
}

func TestListDueColumnDoesNotAlarmOnClosedIssues(t *testing.T) {
	t.Parallel()

	var dueCell func(m model, i jira.Issue, a, b bool) string
	for _, col := range listColumns {
		if col.header == "DUE" {
			dueCell = col.cell
			break
		}
	}
	if dueCell == nil {
		t.Fatal("no DUE column found")
	}

	m := newTabModel([]Tab{{id: 0, baseView: listView}}, 0)
	m.columnWidths = ui.CalculateColumnWidths(120)
	overdue := jira.Issue{Key: "DEV-1", Status: "Trabajando", DueDate: "2026-01-15"}
	done := jira.Issue{Key: "DEV-2", Status: "Done", DueDate: "2026-01-15"}

	if got := dueCell(m, overdue, false, false); !strings.Contains(got, ui.IconDueOverdue) {
		t.Errorf("open overdue list cell = %q, want the overdue icon", got)
	}
	if got := dueCell(m, done, false, false); strings.Contains(got, ui.IconDueOverdue) {
		t.Errorf("closed list cell = %q, should not alarm", got)
	}
}
