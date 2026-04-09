package ui

import (
	"charm.land/lipgloss/v2"
)

type ColumnWidths struct {
	Type      int
	Key       int
	Summary   int
	Reporter  int
	Status    int
	Assignee  int
	Priority  int
	Cursor    int
	Empty     int
	TimeSpent int
}

func CalculateColumnWidths(terminalWidth int) ColumnWidths {
	minWidth := 80
	availableWidth := max(minWidth, terminalWidth-4)

	fixedWidths := ColumnWidths{
		Type:      4,
		Key:       12,
		Priority:  1,
		Cursor:    2,
		Empty:     1,
		TimeSpent: 8,
	}

	statusWidth := 15
	assigneeWidth := 20
	reporterWidth := 20

	if availableWidth < 100 {
		statusWidth = 10
		assigneeWidth = 15
		reporterWidth = 15
	}

	fixedWidths.Status = statusWidth
	fixedWidths.Assignee = assigneeWidth
	fixedWidths.Reporter = reporterWidth

	fixedTotal := fixedWidths.Cursor + fixedWidths.Type + fixedWidths.Empty +
		fixedWidths.Key + fixedWidths.Priority + fixedWidths.Empty +
		fixedWidths.Status + fixedWidths.Empty +
		fixedWidths.Assignee + fixedWidths.Empty +
		fixedWidths.TimeSpent + fixedWidths.Empty + fixedWidths.Reporter + fixedWidths.Empty

	summaryWidth := max(availableWidth-fixedTotal, 50)

	fixedWidths.Summary = summaryWidth

	return fixedWidths
}

func (c ColumnWidths) TotalWidth() int {
	return c.Cursor + c.Type + c.Empty + c.Key + c.Priority + c.Empty +
		c.Summary + c.Empty + c.Reporter + c.Empty + c.Status + c.Empty + c.Assignee + c.Empty + c.TimeSpent
}

func (c ColumnWidths) RenderKey(text string) string {
	return KeyFieldStyle.Width(c.Key).Render(text)
}

func (c ColumnWidths) RenderSummary(text string) string {
	return SummaryFieldStyle.Width(c.Summary).Render(text)
}

func (c ColumnWidths) RenderAssignee(text string) string {
	return AssigneeFieldStyle.Width(c.Assignee).Render(text)
}

func (c ColumnWidths) RenderReporter(text string) string {
	return AssigneeFieldStyle.Width(c.Assignee).Render(text)
}

func (c ColumnWidths) RenderTimeSpent(text string) string {
	return TimeSpentFieldStyle.Width(c.TimeSpent).Render(text)
}

func (c ColumnWidths) RenderEmptySpace() string {
	return lipgloss.NewStyle().Width(c.Empty).Render("")
}
