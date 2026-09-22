package ui

import (
	"fmt"
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
)

// TestRenderTagsRespectsBudget is the contract the list-row alignment depends
// on: a chip run must never be wider than the cell it is given.
func TestRenderTagsRespectsBudget(t *testing.T) {
	t.Parallel()

	tagSets := [][]string{
		nil,
		{"wip"},
		{"urgent", "review"},
		{"backend", "needs-review", "waiting-on-ops"},
		{"a", "b", "c", "d", "e", "f", "g", "h"},
		{strings.Repeat("x", 32)},
	}

	for _, tags := range tagSets {
		for budget := range 60 {
			for _, dimmed := range []bool{false, true} {
				got := RenderTags(tags, budget, dimmed)
				if w := lipgloss.Width(got); w > budget {
					t.Fatalf("RenderTags(%v, budget=%d, dimmed=%v) is %d cells wide, over budget: %q",
						tags, budget, dimmed, w, got)
				}
			}
		}
	}
}

func TestRenderTagsMarksHiddenTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		tags       []string
		budget     int
		wantChips  []string
		wantMarker string
	}{
		{
			name:      "everything fits",
			tags:      []string{"wip", "api"},
			budget:    40,
			wantChips: []string{"#wip", "#api"},
		},
		{
			name:       "one fits, the rest are counted",
			tags:       []string{"wip", "api", "ops"},
			budget:     8,
			wantChips:  []string{"#wip"},
			wantMarker: "+2",
		},
		{
			name:   "nothing fits",
			tags:   []string{"needs-review"},
			budget: 4,
		},
		{
			name:   "zero budget",
			tags:   []string{"wip"},
			budget: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := RenderTags(tt.tags, tt.budget, false)
			for _, chip := range tt.wantChips {
				if !strings.Contains(got, chip) {
					t.Errorf("RenderTags(%v, %d) = %q, want it to contain %q", tt.tags, tt.budget, got, chip)
				}
			}
			if tt.wantMarker != "" && !strings.Contains(got, tt.wantMarker) {
				t.Errorf("RenderTags(%v, %d) = %q, want the overflow marker %q", tt.tags, tt.budget, got, tt.wantMarker)
			}
			if tt.wantMarker == "" && len(tt.wantChips) == 0 && got != "" {
				t.Errorf("RenderTags(%v, %d) = %q, want empty", tt.tags, tt.budget, got)
			}
			if w := lipgloss.Width(got); w > tt.budget {
				t.Errorf("RenderTags(%v, %d) is %d cells wide", tt.tags, tt.budget, w)
			}
		})
	}
}

func TestRenderTagsMarkerCountIsCorrect(t *testing.T) {
	t.Parallel()

	tags := []string{"aa", "bb", "cc", "dd"}
	got := RenderTags(tags, 9, false)

	shown := strings.Count(got, "#")
	hidden := len(tags) - shown
	if hidden > 0 && !strings.Contains(got, fmt.Sprintf("+%d", hidden)) {
		t.Errorf("RenderTags(%v, 9) = %q, showed %d chips so it should say +%d", tags, got, shown, hidden)
	}
}
