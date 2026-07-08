package ui

import (
	"strings"
	"testing"
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
