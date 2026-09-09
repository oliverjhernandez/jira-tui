package ui

import (
	"strings"
	"testing"
	"time"

	"charm.land/lipgloss/v2"
)

func TestFormatTimeSpent(t *testing.T) {
	tests := []struct {
		name    string
		seconds int
		want    string
	}{
		{"zero", 0, "-"},
		{"whole hours", 3600, "1h"},
		{"hours and minutes", 5400, "1h 30m"},
		{"minutes only", 120, "2m"},
		{"less than a minute", 59, "-"},
		{"multiple hours", 7200, "2h"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatTimeSpent(tt.seconds); got != tt.want {
				t.Errorf("FormatTimeSpent(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestTruncateLongString(t *testing.T) {
	tests := []struct {
		name string
		s    string
		max  int
		want string
	}{
		{"shorter than max", "hello", 10, "hello"},
		{"equal to max", "hello", 5, "hello"},
		{"longer than max", "hello world", 5, "hell…"},
		{"zero max", "hello", 0, ""},
		{"negative max", "hello", -3, ""},
		{"unicode runes", "áéíóú", 3, "áé…"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := TruncateLongString(tt.s, tt.max); got != tt.want {
				t.Errorf("TruncateLongString(%q, %d) = %q, want %q", tt.s, tt.max, got, tt.want)
			}
		})
	}
}

func TestGetModalWidthHeight(t *testing.T) {
	if got := GetModalWidth(100, 0.2); got != 20 {
		t.Errorf("GetModalWidth(100, 0.2) = %d, want 20", got)
	}
	if got := GetModalHeight(50, 0.3); got != 15 {
		t.Errorf("GetModalHeight(50, 0.3) = %d, want 15", got)
	}
	if got := GetModalWidth(0, 0.5); got != 0 {
		t.Errorf("GetModalWidth(0, 0.5) = %d, want 0", got)
	}
}

func TestRepeatChar(t *testing.T) {
	if got := RepeatChar("─", 3); got != "───" {
		t.Errorf("RepeatChar(─, 3) = %q, want %q", got, "───")
	}
	if got := RepeatChar("x", 0); got != "" {
		t.Errorf("RepeatChar(x, 0) = %q, want empty", got)
	}
}

func TestOsc8(t *testing.T) {
	got := Osc8("https://example.com", "link")
	if !strings.Contains(got, "https://example.com") {
		t.Errorf("Osc8 output missing url: %q", got)
	}
	if !strings.Contains(got, "link") {
		t.Errorf("Osc8 output missing text: %q", got)
	}
	if !strings.HasPrefix(got, "\x1b]8;;") {
		t.Errorf("Osc8 output missing OSC8 prefix: %q", got)
	}
}

func TestRenderStatusBadge(t *testing.T) {
	tests := []struct {
		name   string
		status string
		want   string // substring expected in output
	}{
		{"in progress", "In Progress", "In Progress"},
		{"done", "Done", "Done"},
		{"selected for development is shortened", "Selected for Development", "Selected"},
		{"ready to deploy is shortened", "Ready to Deploy", "Ready"},
		{"unknown falls through", "Weird Status", "Weird Status"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RenderStatusBadge(tt.status)
			if !strings.Contains(got, tt.want) {
				t.Errorf("RenderStatusBadge(%q) = %q, want it to contain %q", tt.status, got, tt.want)
			}
		})
	}
}

func TestRenderPriority(t *testing.T) {
	// with text should embed the priority name
	got := RenderPriority("High", true)
	if !strings.Contains(got, "High") {
		t.Errorf("RenderPriority(High, true) = %q, want it to contain High", got)
	}

	// without text should still render (icon only) and not panic on unknown
	if RenderPriority("Totally Unknown", false) == "" {
		t.Errorf("RenderPriority(unknown, false) returned empty string")
	}
}

func TestRenderIssueType(t *testing.T) {
	got := RenderIssueType("Bug", true)
	if !strings.Contains(got, "Bug") {
		t.Errorf("RenderIssueType(Bug, true) = %q, want it to contain Bug", got)
	}
	if RenderIssueType("Whatever", false) == "" {
		t.Errorf("RenderIssueType(unknown, false) returned empty string")
	}
}

func TestRenderPanelWithLabel(t *testing.T) {
	// Should not panic for narrow widths where the label exceeds the width.
	out := RenderPanelWithLabel("A very long label that exceeds width", "content", 10, 5, true)
	if out == "" {
		t.Errorf("RenderPanelWithLabel returned empty output")
	}
}

func TestFormatDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
	}{
		{"2026-09-08", "Sep 08"},
		{"", DateUnset},
		{"not-a-date", "not-a-date"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			if got := FormatDate(tt.in); got != tt.want {
				t.Errorf("FormatDate(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestClassifyDue(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 9, 15, 30, 0, 0, time.UTC)

	tests := []struct {
		name string
		in   string
		want DueUrgency
	}{
		{"no due date", "", DueUnset},
		{"unparseable", "not-a-date", DueUnset},
		{"long overdue", "2026-08-01", DueOverdue},
		{"yesterday is overdue", "2026-09-08", DueOverdue},
		{"today", "2026-09-09", DueToday},
		{"tomorrow is soon", "2026-09-10", DueSoon},
		{"the last soon day", "2026-09-12", DueSoon},
		{"one day past soon", "2026-09-13", DueLater},
		{"far out", "2027-01-01", DueLater},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifyDue(tt.in, now); got != tt.want {
				t.Errorf("ClassifyDue(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func TestClassifyDueIgnoresTimeOfDay(t *testing.T) {
	t.Parallel()

	// A due date is "today" all day, not only until the clock passes midnight
	// of the due date's zero hour.
	for _, hour := range []int{0, 9, 23} {
		now := time.Date(2026, 9, 9, hour, 59, 0, 0, time.UTC)
		if got := ClassifyDue("2026-09-09", now); got != DueToday {
			t.Errorf("at %02d:59 ClassifyDue = %v, want today", hour, got)
		}
	}
}

func TestRenderDueIconAndColor(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		in       string
		wantIcon string
		wantText string
	}{
		{"overdue", "2026-09-01", IconDueOverdue, "Sep 01"},
		{"today", "2026-09-09", IconDueToday, "Sep 09"},
		{"soon", "2026-09-11", IconDueSoon, "Sep 11"},
		{"later", "2026-10-30", IconDueLater, "Oct 30"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := RenderDue(tt.in, now)
			if !strings.Contains(got, tt.wantIcon) {
				t.Errorf("RenderDue(%q) = %q, want icon %q", tt.in, got, tt.wantIcon)
			}
			if !strings.Contains(got, tt.wantText) {
				t.Errorf("RenderDue(%q) = %q, want text %q", tt.in, got, tt.wantText)
			}
		})
	}

	icons := map[string]bool{}
	for _, tt := range tests {
		icons[tt.wantIcon] = true
	}
	if len(icons) != len(tests) {
		t.Errorf("each urgency needs its own icon, got %d distinct for %d buckets", len(icons), len(tests))
	}
}

func TestRenderDueWithoutADate(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	got := RenderDue("", now)

	if !strings.Contains(got, DateUnset) {
		t.Errorf("RenderDue(\"\") = %q, want the unset placeholder", got)
	}
	for _, icon := range []string{IconDueOverdue, IconDueToday, IconDueSoon, IconDueLater} {
		if strings.Contains(got, icon) {
			t.Errorf("RenderDue(\"\") = %q, should carry no urgency icon", got)
		}
	}
}

func TestColumnRenderDueDateFitsItsWidth(t *testing.T) {
	t.Parallel()

	c := CalculateColumnWidths(120)
	now := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)

	for _, iso := range []string{"", "2026-09-01", "2026-09-09", "2026-09-11", "2026-12-31"} {
		cell := c.RenderDueDate(iso, now)
		if got := lipgloss.Width(cell); got != c.DueDate {
			t.Errorf("RenderDueDate(%q) width = %d, want %d (%q)", iso, got, c.DueDate, cell)
		}
	}
}

func TestRenderPriorityUnsetIsNotAlarming(t *testing.T) {
	t.Parallel()

	alarming := []string{IconError, IconPriorityCritical, IconPriorityHighest, IconPriorityHigh}

	unset := []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"whitespace", "   "},
		{"por definir", "Por Definir"},
		{"sin definir", "Sin definir"},
		{"none", "None"},
		{"unrecognized label", "Totally Unknown"},
	}

	for _, tt := range unset {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, showText := range []bool{false, true} {
				got := RenderPriority(tt.in, showText)
				if !strings.Contains(got, IconPriorityUnset) {
					t.Errorf("RenderPriority(%q, %v) = %q, want the unset dash", tt.in, showText, got)
				}
				for _, icon := range alarming {
					if icon != "" && strings.Contains(got, icon) {
						t.Errorf("RenderPriority(%q, %v) = %q, carries the alarming icon %q",
							tt.in, showText, got, icon)
					}
				}
			}
		})
	}
}

func TestRenderPriorityUnsetOmitsTheEmptyLabel(t *testing.T) {
	t.Parallel()

	// An issue with no priority must not render "- " with a dangling separator.
	for _, in := range []string{"", "  "} {
		got := RenderPriority(in, true)
		if strings.TrimSpace(stripStyle(got)) != IconPriorityUnset {
			t.Errorf("RenderPriority(%q, true) = %q, want just the dash", in, stripStyle(got))
		}
	}
}

func TestRenderPriorityKeepsKnownLevels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in       string
		wantIcon string
	}{
		{"Critica", IconPriorityCritical},
		{"Highest", IconPriorityHighest},
		{"High", IconPriorityHigh},
		{"Medium", IconPriorityMedium},
		{"Low", IconPriorityLow},
		{"Lowest", IconPriorityLowest},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			t.Parallel()
			got := RenderPriority(tt.in, true)
			if !strings.Contains(got, tt.wantIcon) {
				t.Errorf("RenderPriority(%q) = %q, want icon %q", tt.in, got, tt.wantIcon)
			}
			if !strings.Contains(got, tt.in) {
				t.Errorf("RenderPriority(%q) = %q, want the label kept", tt.in, got)
			}
			if strings.Contains(got, IconPriorityUnset) {
				t.Errorf("RenderPriority(%q) = %q, should not read as unset", tt.in, got)
			}
		})
	}
}

func stripStyle(s string) string {
	var b strings.Builder
	inEscape := false
	for _, r := range s {
		switch {
		case r == '\x1b':
			inEscape = true
		case inEscape && r == 'm':
			inEscape = false
		case !inEscape:
			b.WriteRune(r)
		}
	}
	return b.String()
}
