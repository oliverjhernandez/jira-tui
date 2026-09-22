package main

import (
	"strings"
	"testing"
	"time"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func TestValidateDate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		wantErr bool
	}{
		{"empty clears the field", "", false},
		{"iso date", "2026-09-08", false},
		{"display format rejected", "Sep 08", true},
		{"day first rejected", "08-09-2026", true},
		{"impossible date rejected", "2026-02-31", true},
		{"garbage rejected", "tomorrow", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateDate(tt.in)
			if tt.wantErr && err == nil {
				t.Errorf("validateDate(%q) = nil, want an error", tt.in)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("validateDate(%q) = %v, want nil", tt.in, err)
			}
		})
	}
}

func TestResolveStartDateField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		fields []jira.Field
		want   string
	}{
		{
			name:   "english name",
			fields: []jira.Field{{ID: "duedate", Name: "Due date"}, {ID: "customfield_10015", Name: "Start date"}},
			want:   "customfield_10015",
		},
		{
			name:   "spanish name",
			fields: []jira.Field{{ID: "customfield_10500", Name: "Fecha de inicio"}},
			want:   "customfield_10500",
		},
		{
			name:   "case and spacing insensitive",
			fields: []jira.Field{{ID: "customfield_10015", Name: "  START DATE "}},
			want:   "customfield_10015",
		},
		{
			name:   "exact name wins over the loose one",
			fields: []jira.Field{{ID: "customfield_1", Name: "Start"}, {ID: "customfield_2", Name: "Start date"}},
			want:   "customfield_2",
		},
		{
			name:   "absent field yields no id",
			fields: []jira.Field{{ID: "duedate", Name: "Due date"}},
			want:   "",
		},
		{
			name:   "no fields at all",
			fields: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveStartDateField(tt.fields); got != tt.want {
				t.Errorf("resolveStartDateField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMetadataPanelShowsDates(t *testing.T) {
	t.Parallel()

	dueDate := time.Now().AddDate(0, 0, 30).Format(ui.DateLayout)

	m := newTabModel([]Tab{{id: 0, baseView: detailView}}, 0)
	m.mode = detailView
	m.activeIssue = &jira.Issue{
		Key:       "DEV-1",
		Type:      "Task",
		Summary:   "Fix the thing",
		Status:    "In Progress",
		Assignee:  "Oliver Hernandez",
		StartDate: "2026-09-01",
		DueDate:   dueDate,
	}

	panel := m.renderMetadataPanel(120, 7)
	lines := strings.Split(panel, "\n")

	for _, want := range []string{"Due", ui.FormatDate(dueDate), "Start", "Sep 01", "Type"} {
		if !strings.Contains(panel, want) {
			t.Errorf("metadata panel missing %q:\n%s", want, panel)
		}
	}

	if len(lines) > 8 {
		t.Errorf("metadata panel grew to %d lines, want no more than 8", len(lines))
	}

	var dueLine, startLine, typeLine int
	for i, ln := range lines {
		switch {
		case strings.Contains(ln, "Due"):
			dueLine = i
		case strings.Contains(ln, "Sep 01"):
			startLine = i
		case strings.Contains(ln, "Type"):
			typeLine = i
		}
	}
	if dueLine >= startLine || startLine >= typeLine {
		t.Errorf("rows out of order: due=%d start=%d type=%d\n%s", dueLine, startLine, typeLine, panel)
	}
}

func TestMetadataPanelDueDateIsNotDimmed(t *testing.T) {
	t.Parallel()

	m := newTabModel([]Tab{{id: 0, baseView: detailView}}, 0)
	m.mode = detailView
	m.activeIssue = &jira.Issue{
		Key:     "DEV-1",
		Type:    "Task",
		Status:  "In Progress",
		DueDate: time.Now().AddDate(0, 0, 30).Format(ui.DateLayout),
	}

	panel := m.renderMetadataPanel(120, 7)

	if !strings.Contains(panel, ui.IconDueLater) {
		t.Errorf("far-off due date should carry the calendar icon:\n%s", panel)
	}
	if strings.Contains(panel, ui.DimTextStyle.Render(ui.FormatDate(m.activeIssue.DueDate))) {
		t.Errorf("due date is still rendered dim:\n%s", panel)
	}
}

func TestOpenDatesFormPrefillsAndTargetsTheIssue(t *testing.T) {
	t.Parallel()

	m := newTabModel([]Tab{{id: 0, baseView: detailView}}, 0)
	m.mode = detailView
	m.startDateFieldID = "customfield_10015"
	m.activeIssue = &jira.Issue{Key: "DEV-1", StartDate: "2026-09-01", DueDate: "2026-09-12"}

	next, _ := m.Update(keyPress("D"))
	dm := next.(model)

	if dm.mode != datesView {
		t.Fatalf("D should open the dates form, mode = %v", dm.mode)
	}
	if dm.datesData == nil {
		t.Fatal("dates form data not created")
	}
	if dm.datesData.StartDate != "2026-09-01" || dm.datesData.DueDate != "2026-09-12" {
		t.Errorf("form not prefilled: start=%q due=%q", dm.datesData.StartDate, dm.datesData.DueDate)
	}
	if dm.pendingIssue == nil || dm.pendingIssue.Key != "DEV-1" {
		t.Errorf("pending issue = %+v, want DEV-1", dm.pendingIssue)
	}

	next, _ = dm.Update(keyPress("esc"))
	em := next.(model)
	if em.mode != detailView || em.datesData != nil {
		t.Errorf("esc should close the form: mode = %v, data = %+v", em.mode, em.datesData)
	}
}

func TestDatesFormWithoutStartFieldEditsDueOnly(t *testing.T) {
	t.Parallel()

	withStart := NewDatesFormData("2026-09-01", "2026-09-12", true)
	withStart.Form.Init()
	if v := withStart.Form.View(); !strings.Contains(v, "Start date") || !strings.Contains(v, "Due date") {
		t.Errorf("both inputs expected when the field is known:\n%s", v)
	}

	dueOnly := NewDatesFormData("", "2026-09-12", false)
	dueOnly.Form.Init()
	v := dueOnly.Form.View()
	if strings.Contains(v, "Start date") {
		t.Errorf("start input shown even though the field is unknown:\n%s", v)
	}
	if !strings.Contains(v, "Due date") {
		t.Errorf("due input missing:\n%s", v)
	}
}

func TestDatesFormOpensWithoutStartField(t *testing.T) {
	t.Parallel()

	m := newTabModel([]Tab{{id: 0, baseView: detailView}}, 0)
	m.mode = detailView
	m.startDateFieldID = ""
	m.activeIssue = &jira.Issue{Key: "DEV-1", DueDate: "2026-09-12"}

	next, _ := m.Update(keyPress("D"))
	dm := next.(model)

	if dm.mode != datesView {
		t.Fatalf("D should still open the form for the due date, mode = %v", dm.mode)
	}
	if dm.statusMessage.content == "" {
		t.Error("expected a message explaining that start date is unavailable")
	}
}

func TestFieldsLoadedResolvesStartField(t *testing.T) {
	t.Parallel()

	m := newTabModel([]Tab{{id: 0, baseView: listView}}, 0)
	m.loadingCount = 1

	next, _ := m.Update(fieldsLoadedMsg{fields: []jira.Field{
		{ID: "customfield_10015", Name: "Start date"},
	}})
	nm := next.(model)

	if nm.startDateFieldID != "customfield_10015" {
		t.Errorf("startDateFieldID = %q, want customfield_10015", nm.startDateFieldID)
	}
	if nm.loadingCount != 0 {
		t.Errorf("loadingCount = %d, want 0", nm.loadingCount)
	}
}

func TestResolveFieldByNames(t *testing.T) {
	t.Parallel()

	fields := []jira.Field{
		{ID: "customfield_10021", Name: "Flagged"},
		{ID: "customfield_10485", Name: "Motivo de bloqueo"},
		{ID: "customfield_10999", Name: " Start Date "},
	}

	tests := []struct {
		name  string
		names []string
		want  string
	}{
		{"exact match", []string{"flagged"}, "customfield_10021"},
		{"case insensitive", []string{"FLAGGED"}, "customfield_10021"},
		{"trims whitespace", []string{"start date"}, "customfield_10999"},
		{"first name wins", []string{"motivo de bloqueo", "flagged"}, "customfield_10485"},
		{"falls through to later name", []string{"nope", "flagged"}, "customfield_10021"},
		{"no match", []string{"nonexistent"}, ""},
		{"empty candidates", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := resolveFieldByNames(fields, tt.names); got != tt.want {
				t.Errorf("resolveFieldByNames(%v) = %q, want %q", tt.names, got, tt.want)
			}
		})
	}
}
