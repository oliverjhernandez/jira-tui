package ui

import "testing"

func TestCalculateColumnWidths(t *testing.T) {
	tests := []struct {
		name          string
		terminalWidth int
	}{
		{"narrow terminal", 60},
		{"exactly minimum", 80},
		{"just under 100", 99},
		{"wide terminal", 200},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cw := CalculateColumnWidths(tt.terminalWidth)

			if cw.Summary < 50 {
				t.Errorf("Summary width = %d, want at least 50", cw.Summary)
			}
			if cw.Type <= 0 || cw.Key <= 0 || cw.Status <= 0 {
				t.Errorf("fixed widths must be positive: %+v", cw)
			}
			if cw.TotalWidth() <= 0 {
				t.Errorf("TotalWidth() = %d, want positive", cw.TotalWidth())
			}
		})
	}
}

func TestCalculateColumnWidthsNarrowShrinksColumns(t *testing.T) {
	narrow := CalculateColumnWidths(90) // availableWidth < 100 branch
	wide := CalculateColumnWidths(200)

	if narrow.Status >= wide.Status {
		t.Errorf("narrow Status (%d) should be smaller than wide Status (%d)", narrow.Status, wide.Status)
	}
	if narrow.Reporter >= wide.Reporter {
		t.Errorf("narrow Reporter (%d) should be smaller than wide Reporter (%d)", narrow.Reporter, wide.Reporter)
	}
}

func TestTotalWidthSumsComponents(t *testing.T) {
	cw := ColumnWidths{
		Type: 1, Key: 2, Summary: 3, Reporter: 4, Status: 5,
		Assignee: 6, DueDate: 7, CreatedDate: 8, Priority: 9,
		Cursor: 10, Empty: 1, TimeSpent: 11,
	}
	// TotalWidth adds Empty (=1) seven times plus the listed non-empty columns
	// (Assignee is intentionally not part of the total).
	want := cw.Cursor + cw.Type + cw.Key + cw.Priority + cw.Summary +
		cw.Reporter + cw.Status + cw.DueDate + cw.CreatedDate + cw.TimeSpent + (cw.Empty * 7)
	if got := cw.TotalWidth(); got != want {
		t.Errorf("TotalWidth() = %d, want %d", got, want)
	}
}
