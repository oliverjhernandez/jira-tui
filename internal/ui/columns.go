package ui

type ColumnWidths struct {
	Type        int
	Key         int
	Summary     int
	Reporter    int
	Status      int
	Assignee    int
	DueDate     int
	CreatedDate int
	Priority    int
	Cursor      int
	Empty       int
	TimeSpent   int
}

func CalculateColumnWidths(terminalWidth int) ColumnWidths {
	minWidth := 80
	availableWidth := max(minWidth, terminalWidth-PanelOverheadWidth)

	fixedWidths := ColumnWidths{
		Type:      4,
		Key:       12,
		Priority:  3, // wide enough for the "PRI" header label
		Cursor:    2,
		Empty:     1,
		TimeSpent: 8,
		// Status renders via RenderStatusBadge, which pads to the static
		// ColWidthStatus; reserve the same here so the layout math is honest.
		Status: ColWidthStatus,
	}

	assigneeWidth := 20
	dueDateWidth := 10
	createdDateWidth := 10
	reporterWidth := 20

	if availableWidth < 100 {
		assigneeWidth = 15
		reporterWidth = 15
	}

	fixedWidths.Assignee = assigneeWidth
	fixedWidths.DueDate = dueDateWidth
	fixedWidths.CreatedDate = createdDateWidth
	fixedWidths.Reporter = reporterWidth

	// The list row is: cursor prefix + 10 columns joined by single-space gaps
	// (9 gaps). Summary takes whatever is left.
	const gaps = 9
	fixedTotal := fixedWidths.Cursor +
		fixedWidths.Type + fixedWidths.Key + fixedWidths.Status + fixedWidths.Priority +
		fixedWidths.Reporter + fixedWidths.Assignee + fixedWidths.CreatedDate +
		fixedWidths.DueDate + fixedWidths.TimeSpent +
		fixedWidths.Empty*gaps

	summaryWidth := max(availableWidth-fixedTotal, 50)

	fixedWidths.Summary = summaryWidth

	return fixedWidths
}

// TotalWidth is the full rendered row width: the cursor prefix, all ten
// columns, and the nine single-space gaps between them.
func (c ColumnWidths) TotalWidth() int {
	const gaps = 9
	return c.Cursor +
		c.Type + c.Key + c.Status + c.Priority + c.Summary +
		c.Reporter + c.Assignee + c.CreatedDate + c.DueDate + c.TimeSpent +
		c.Empty*gaps
}

func (c ColumnWidths) RenderKey(text string) string {
	return KeyFieldStyle.Width(c.Key).Render(text)
}

func (c ColumnWidths) RenderSummary(text string, selected bool, dimmed bool) string {
	if selected {
		return SummaryFieldSelectedStyle.Width(c.Summary).Render(text)
	}
	if dimmed {
		return DimTextStyle.Width(c.Summary).Render(text)
	}
	return SummaryFieldStyle.Width(c.Summary).Render(text)
}

func (c ColumnWidths) RenderDueDate(text string) string {
	return DueDateFieldStyle.Width(c.DueDate).Render(TruncateLongString(text, c.DueDate))
}

func (c ColumnWidths) RenderCreatedDate(text string) string {
	return CreatedDateFieldStyle.Width(c.CreatedDate).Render(TruncateLongString(text, c.CreatedDate))
}

func (c ColumnWidths) RenderReporter(text string) string {
	return ReporterFieldStyle.Width(c.Reporter).Render(TruncateLongString(text, c.Reporter))
}

func (c ColumnWidths) RenderAssignee(text string) string {
	return AssigneeFieldStyle.Width(c.Assignee).Render(TruncateLongString(text, c.Assignee))
}

func (c ColumnWidths) RenderTimeSpent(text string) string {
	return TimeSpentFieldStyle.Width(c.TimeSpent).Render(text)
}
