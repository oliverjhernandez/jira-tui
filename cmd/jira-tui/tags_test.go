package main

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type fakeTagStore struct {
	tags map[string][]string
	err  error
}

func newFakeTagStore(seed map[string][]string) *fakeTagStore {
	if seed == nil {
		seed = map[string][]string{}
	}
	return &fakeTagStore{tags: seed}
}

func (f *fakeTagStore) Tags(issueKey string) []string { return f.tags[issueKey] }

func (f *fakeTagStore) SetTags(_ context.Context, issueKey string, tags []string) error {
	if f.err != nil {
		return f.err
	}
	if len(tags) == 0 {
		delete(f.tags, issueKey)
		return nil
	}
	f.tags[issueKey] = tags
	return nil
}

func (f *fakeTagStore) AllTags() []string {
	var all []string
	for _, tags := range f.tags {
		all = append(all, tags...)
	}
	return all
}

func tagsModel(t *testing.T, store *fakeTagStore, issues ...jira.Issue) model {
	t.Helper()
	m := filteringModel(t, "", issues...)
	m.tags = store
	return m
}

func TestTagsFromListTargetsTheSelectedIssue(t *testing.T) {
	t.Parallel()

	m := tagsModel(t, newFakeTagStore(nil), issue("DEV-1", "alpha"), issue("DEV-2", "bravo"))
	m.selectedIssue = &m.issues[1]

	next, _ := m.Update(keyPress("#"))
	tm := next.(model)

	if tm.mode != tagsView {
		t.Fatalf("mode = %v, want tagsView", tm.mode)
	}
	if tm.tagsData == nil {
		t.Fatal("tagsData should be populated")
	}
	if tm.tagsData.IssueKey != "DEV-2" {
		t.Errorf("tags would be applied to %s, but the cursor is on DEV-2", tm.tagsData.IssueKey)
	}
}

// TestTagsFromDetailTargetsTheOpenIssue is the wrong-issue guard: the list
// selection and the open issue deliberately differ.
func TestTagsFromDetailTargetsTheOpenIssue(t *testing.T) {
	t.Parallel()

	m := tagsModel(t, newFakeTagStore(nil), issue("DEV-1", "alpha"), issue("DEV-2", "bravo"))
	m.selectedIssue = &m.issues[1]
	m.activeIssue = &m.issues[0]
	m.mode = detailView
	m.baseView = detailView

	next, _ := m.Update(keyPress("#"))
	tm := next.(model)

	if tm.tagsData == nil {
		t.Fatal("tagsData should be populated")
	}
	if tm.tagsData.IssueKey != "DEV-1" {
		t.Errorf("tags would be applied to %s, but the detail view is showing DEV-1", tm.tagsData.IssueKey)
	}
	if tm.previousMode != detailView {
		t.Errorf("previousMode = %v, want detailView so esc returns to the issue", tm.previousMode)
	}
}

// TestTagsSurviveARefreshDuringEditing is why TagsFormData holds a key string
// rather than an issue pointer: a poll can rebuild m.sections mid-edit.
func TestTagsSurviveARefreshDuringEditing(t *testing.T) {
	t.Parallel()

	store := newFakeTagStore(nil)
	m := tagsModel(t, store, issue("DEV-1", "alpha"), issue("DEV-2", "bravo"))
	m.selectedIssue = &m.issues[1]

	next, _ := m.Update(keyPress("#"))
	tm := next.(model)

	tm.issues = []jira.Issue{issue("DEV-9", "new"), issue("DEV-2", "bravo"), issue("DEV-1", "alpha")}
	tm.sections = tm.sectionsFor(tm.issues)
	tm.rebuildFilteredSections()
	tm.selectedIssue = &tm.issues[0]

	tm.tagsData.Raw = "urgent"
	tm.tagsData.Form.State = huh.StateCompleted
	done, _ := tm.updateTagsView(nil)
	dm := done.(model)

	if got, want := store.tags["DEV-2"], []string{"urgent"}; !reflect.DeepEqual(got, want) {
		t.Errorf("DEV-2 tags = %v, want %v: the write followed the refreshed selection instead of the issue being edited", got, want)
	}
	if len(store.tags["DEV-9"]) != 0 {
		t.Errorf("DEV-9 should not have been tagged, got %v", store.tags["DEV-9"])
	}
	if dm.mode != listView {
		t.Errorf("mode = %v, want listView", dm.mode)
	}
}

// TestTagsNeverTouchLoadingCount pins the deliberate break from the
// postXxxCmd convention: a leaked count silently disables ctrl+r.
func TestTagsNeverTouchLoadingCount(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		finish func(m model) model
	}{
		{
			name: "cancelled with esc",
			finish: func(m model) model {
				next, _ := m.updateTagsView(keyPress("esc"))
				return next.(model)
			},
		},
		{
			name: "submitted",
			finish: func(m model) model {
				m.tagsData.Raw = "urgent"
				m.tagsData.Form.State = huh.StateCompleted
				next, _ := m.updateTagsView(nil)
				return next.(model)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			m := tagsModel(t, newFakeTagStore(nil), issue("DEV-1", "alpha"))
			m.selectedIssue = &m.issues[0]

			next, _ := m.Update(keyPress("#"))
			opened := next.(model)
			if opened.loadingCount != 0 {
				t.Fatalf("opening the tag editor set loadingCount to %d", opened.loadingCount)
			}

			if got := tt.finish(opened).loadingCount; got != 0 {
				t.Errorf("loadingCount = %d, want 0: tags are a local write with no request in flight", got)
			}
		})
	}
}

func TestTagWriteErrorSurfacesAndClosesTheModal(t *testing.T) {
	t.Parallel()

	store := newFakeTagStore(nil)
	store.err = errors.New("disk on fire")

	m := tagsModel(t, store, issue("DEV-1", "alpha"))
	m.selectedIssue = &m.issues[0]

	next, _ := m.Update(keyPress("#"))
	tm := next.(model)
	tm.tagsData.Raw = "urgent"
	tm.tagsData.Form.State = huh.StateCompleted

	done, _ := tm.updateTagsView(nil)
	dm := done.(model)

	if dm.mode == tagsView {
		t.Error("a failed write should still close the modal, not trap the user in the form")
	}
	if dm.statusMessage.msgType != errStatusBarMsg {
		t.Errorf("status message type = %v, want an error", dm.statusMessage.msgType)
	}
}

func TestClearingTagsRemovesThem(t *testing.T) {
	t.Parallel()

	store := newFakeTagStore(map[string][]string{"DEV-1": {"urgent"}})
	m := tagsModel(t, store, issue("DEV-1", "alpha"))
	m.selectedIssue = &m.issues[0]

	next, _ := m.Update(keyPress("#"))
	tm := next.(model)
	if tm.tagsData.Raw != "urgent" {
		t.Errorf("the form should prefill the current tags, got %q", tm.tagsData.Raw)
	}

	tm.tagsData.Raw = "  "
	tm.tagsData.Form.State = huh.StateCompleted
	done, _ := tm.updateTagsView(nil)

	if got := store.tags["DEV-1"]; got != nil {
		t.Errorf("an empty field should clear the tags, got %v", got)
	}
	if dm := done.(model); dm.statusMessage.msgType != successStatusBarMsg {
		t.Errorf("status message type = %v, want success", dm.statusMessage.msgType)
	}
}

func TestTagsWithNoStoreDoNotPanic(t *testing.T) {
	t.Parallel()

	m := filteringModel(t, "", issue("DEV-1", "alpha"))
	m.selectedIssue = &m.issues[0]

	assertNoPanic(t, "# with no tag store", func() {
		next, _ := m.Update(keyPress("#"))
		tm := next.(model)
		if tm.mode == tagsView {
			t.Error("the tag editor should not open when the store failed to load")
		}
		if tm.statusMessage.msgType != errStatusBarMsg {
			t.Error("the user should be told tags are unavailable")
		}
	})
}

func TestParseTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		want    []string
		wantErr bool
	}{
		{name: "comma separated", raw: "urgent, review", want: []string{"review", "urgent"}},
		{name: "space separated", raw: "urgent review", want: []string{"review", "urgent"}},
		{name: "mixed separators and padding", raw: " urgent ,,  review ", want: []string{"review", "urgent"}},
		{name: "sigils are stripped", raw: "#urgent #review", want: []string{"review", "urgent"}},
		{name: "case folded and deduped", raw: "Urgent, urgent", want: []string{"urgent"}},
		{name: "empty clears", raw: "   ", want: nil},
		{name: "invalid rune", raw: "urgent, ¡nope!", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseTags(tt.raw)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseTags(%q) = %v, want an error", tt.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseTags(%q) unexpected error: %v", tt.raw, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parseTags(%q) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

func TestTagFilterRebuildsAfterAWrite(t *testing.T) {
	t.Parallel()

	store := newFakeTagStore(map[string][]string{"DEV-1": {"urgent"}, "DEV-2": {"urgent"}})
	m := tagsModel(t, store, issue("DEV-1", "alpha"), issue("DEV-2", "bravo"))
	m.filtering = true
	m.textInput.SetValue("#urgent")
	m.rebuildFilteredSections()

	if got := countFilteredIssues(m); got != 2 {
		t.Fatalf("the #urgent filter should start with 2 issues, got %d", got)
	}

	// The editor is opened directly rather than with a keypress: while the
	// filter prompt has focus, "#" is filter input, not a binding.
	m.selectedIssue = &m.issues[1]
	m.previousMode = m.mode
	m.mode = tagsView
	m.tagsData = NewTagsFormData("DEV-2", m.tagsOf("DEV-2"), m.knownTags())
	m.tagsData.Raw = ""
	m.tagsData.Form.State = huh.StateCompleted

	done, _ := m.updateTagsView(nil)

	if got := countFilteredIssues(done.(model)); got != 1 {
		t.Errorf("after clearing DEV-2's tags the #urgent filter should hold 1 issue, got %d", got)
	}
}

// TestFilterPromptSwallowsTheTagKey pins the binding's placement: "#" must
// reach the filter textinput while the prompt has focus, or "/#urgent" is
// untypeable.
func TestFilterPromptSwallowsTheTagKey(t *testing.T) {
	t.Parallel()

	m := tagsModel(t, newFakeTagStore(nil), issue("DEV-1", "alpha"))
	m.selectedIssue = &m.issues[0]

	opened, _ := m.Update(keyPress("/"))
	next, _ := opened.(model).Update(keyPress("#"))
	tm := next.(model)

	if tm.mode == tagsView {
		t.Error("# opened the tag editor instead of typing into the active filter")
	}
	if !strings.Contains(tm.textInput.Value(), "#") {
		t.Errorf("the filter should have received the #, got %q", tm.textInput.Value())
	}
}

func TestTagWriteKeepsTheFilterSentinelNil(t *testing.T) {
	t.Parallel()

	m := tagsModel(t, newFakeTagStore(nil), issue("DEV-1", "alpha"))
	m.selectedIssue = &m.issues[0]

	next, _ := m.Update(keyPress("#"))
	tm := next.(model)
	tm.tagsData.Raw = "urgent"
	tm.tagsData.Form.State = huh.StateCompleted
	done, _ := tm.updateTagsView(nil)

	if dm := done.(model); dm.filteredSections != nil {
		t.Error("with no filter active, filteredSections must stay nil: it is the no-filter sentinel")
	}
}

func TestTaggedRowsStayAligned(t *testing.T) {
	t.Parallel()

	store := newFakeTagStore(map[string][]string{
		"DEV-1": {"urgent", "needs-review", "waiting-on-ops", "backend", "frontend"},
	})

	for _, width := range []int{80, 90, 100, 120, 160, 200, 240} {
		m := tagsModel(t, store, issue("DEV-1", "alpha"))
		m.windowWidth = width
		m.columnWidths = ui.CalculateColumnWidths(width)

		for _, selected := range []bool{false, true} {
			row := m.renderIssueRow(m.issues[0], selected, false)
			if got, want := lipgloss.Width(row), m.columnWidths.TotalWidth(); got != want {
				t.Errorf("width=%d selected=%v: tagged row is %d cells, want %d", width, selected, got, want)
			}
		}
	}
}

func TestTagsYieldToTheSummaryOnNarrowTerminals(t *testing.T) {
	t.Parallel()

	store := newFakeTagStore(map[string][]string{"DEV-1": {"urgent"}})
	m := tagsModel(t, store, issue("DEV-1", "a readable summary"))
	m.windowWidth = 80
	m.columnWidths = ui.CalculateColumnWidths(80)

	row := stripANSI(m.renderIssueRow(m.issues[0], false, false))
	if !strings.Contains(row, "a readable summary") {
		t.Errorf("the summary must survive intact on a narrow terminal, got %q", row)
	}
	if strings.Contains(row, "#urgent") {
		t.Errorf("tags should drop out when SUMMARY is already at its floor, got %q", row)
	}
}

func countFilteredIssues(m model) int {
	total := 0
	for _, s := range m.navSections() {
		total += len(s.Issues)
	}
	return total
}
