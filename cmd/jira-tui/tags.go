package main

import (
	"context"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/store"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

// tagStore is the local tag persistence the TUI needs, declared at the consumer
// so tests can substitute a map.
type tagStore interface {
	Tags(issueKey string) []string
	SetTags(ctx context.Context, issueKey string, tags []string) error
	AllTags() []string
}

// TagsFormData holds the tag editor's state. IssueKey is captured as a string
// when the form opens: a key cannot alias into m.sections the way an issue
// pointer does, so a background refresh mid-edit cannot retarget the write.
type TagsFormData struct {
	IssueKey string
	Raw      string
	Form     *huh.Form
}

func NewTagsFormData(issueKey string, current, known []string) *TagsFormData {
	t := &TagsFormData{
		IssueKey: issueKey,
		Raw:      strings.Join(current, ", "),
	}

	description := "comma or space separated; empty clears them"
	if len(known) > 0 {
		description += "\nin use: " + ui.TruncateLongString(strings.Join(known, " "), 60)
	}

	t.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Tags").
				Description(description).
				Placeholder("urgent, review").
				CharLimit(120).
				Value(&t.Raw).
				Validate(validateTagsInput),
		),
	).WithWidth(50)

	return t
}

// parseTags splits user input on commas and whitespace, then hands the pieces
// to the store's normalizer so the UI and the file agree on what a tag is.
func parseTags(raw string) ([]string, error) {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	})
	if len(fields) == 0 {
		return nil, nil
	}
	return store.NormalizeTags(fields)
}

func validateTagsInput(raw string) error {
	_, err := parseTags(raw)
	return err
}

func (m model) tagsOf(issueKey string) []string {
	if m.tags == nil || issueKey == "" {
		return nil
	}
	return m.tags.Tags(issueKey)
}

func (m model) knownTags() []string {
	if m.tags == nil {
		return nil
	}
	return m.tags.AllTags()
}

func (m model) openTagsFor(issue *jira.Issue) (tea.Model, tea.Cmd) {
	if issue == nil {
		return m, nil
	}
	if m.tags == nil {
		m.setErrorMsg("Tags unavailable: the local state file could not be opened")
		return m, m.clearStatusAfter(clearMsgTimeout)
	}

	key := issue.Key
	m.previousMode = m.mode
	m.mode = tagsView
	m.tagsData = NewTagsFormData(key, m.tagsOf(key), m.knownTags())
	return m, m.tagsData.Form.Init()
}

// updateTagsView writes through to the store synchronously. The write is local
// and takes microseconds, so it needs neither a tea.Cmd nor loadingCount, and
// skipping afterIssueAction avoids a pointless board refetch that would
// invalidate the caller's selection.
func (m model) updateTagsView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		if keyPressMsg.String() == "esc" {
			m.mode = m.previousMode
			m.tagsData = nil
			return m, nil
		}
	}

	form, cmd := m.tagsData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.tagsData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.tagsData.Form.State == huh.StateCompleted {
		key := m.tagsData.IssueKey
		tags, err := parseTags(m.tagsData.Raw)
		m.mode = m.previousMode
		m.tagsData = nil

		switch {
		case err != nil:
			m.setError("reading tags", err)
		default:
			if err := m.tags.SetTags(context.Background(), key, tags); err != nil {
				m.setError("saving tags", err)
			} else if len(tags) == 0 {
				m.setSuccess("Tags cleared on " + key)
			} else {
				m.setSuccess(fmt.Sprintf("Tagged %s: %s", key, strings.Join(tags, " ")))
			}
		}

		m.rebuildFilteredSections()
		m.selectIssueByKey(key)
		m.listViewport.SetContent(m.buildListContent())
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderTagsView() string {
	label := "Tags"
	if m.tagsData != nil && m.tagsData.IssueKey != "" {
		label += " " + m.tagsData.IssueKey
	}
	return m.renderModal(label, m.tagsData.Form.View(), 0.3, 0.45)
}

// openTagStore resolves and loads the local state file. A failure disables
// tags rather than stopping the app: the rest of jira-tui does not need them.
func openTagStore() (tagStore, error) {
	path, err := store.DefaultPath()
	if err != nil {
		return nil, err
	}
	s, err := store.Open(path)
	if err != nil {
		return s, err
	}
	return s, nil
}
