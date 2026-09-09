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
