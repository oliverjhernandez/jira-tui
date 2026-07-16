package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

const maxRecentSearches = 10

// issueKeyPattern matches a full Jira issue key like "DEV-123". JQL only allows
// exact key matching, so we add a key clause to the search only when the query
// looks like a key.
var issueKeyPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]+-\d+$`)

type searchCategory int

const (
	catKey searchCategory = iota
	catSummary
	catDescription
)

func (c searchCategory) label() string {
	switch c {
	case catKey:
		return "Key matches"
	case catSummary:
		return "Summary matches"
	default:
		return "Description matches"
	}
}

type searchResult struct {
	issue    jira.Issue
	category searchCategory
}

type searchResultsLoadedMsg struct {
	query   string
	results []searchResult
}

// buildSearchJQL turns a free-text query into a JQL that matches the key,
// summary or description.
func buildSearchJQL(query string) string {
	q := strings.TrimSpace(query)
	esc := strings.ReplaceAll(q, `"`, `\"`)

	clauses := []string{
		fmt.Sprintf(`summary ~ "%s*"`, esc),
		fmt.Sprintf(`description ~ "%s*"`, esc),
	}
	if issueKeyPattern.MatchString(q) {
		clauses = append([]string{fmt.Sprintf(`key = "%s"`, esc)}, clauses...)
	}
	return "(" + strings.Join(clauses, " OR ") + ") ORDER BY updated DESC"
}

// rankSearchResults groups issues by where the query matched — key, then
// summary, then description — preserving the incoming (updated-desc) order
// within each group.
func rankSearchResults(issues []jira.Issue, query string) []searchResult {
	q := strings.ToLower(strings.TrimSpace(query))

	var byKey, bySummary, byDesc []searchResult
	for _, is := range issues {
		switch {
		case q != "" && strings.Contains(strings.ToLower(is.Key), q):
			byKey = append(byKey, searchResult{issue: is, category: catKey})
		case q != "" && strings.Contains(strings.ToLower(is.Summary), q):
			bySummary = append(bySummary, searchResult{issue: is, category: catSummary})
		default:
			byDesc = append(byDesc, searchResult{issue: is, category: catDescription})
		}
	}

	out := make([]searchResult, 0, len(issues))
	out = append(out, byKey...)
	out = append(out, bySummary...)
	out = append(out, byDesc...)
	return out
}

func (m model) searchIssuesCmd(query string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}
		issues, err := m.client.SearchIssuesJql(context.Background(), buildSearchJQL(query))
		if err != nil {
			return errMsg{err}
		}
		return searchResultsLoadedMsg{query: query, results: rankSearchResults(issues, query)}
	}
}

// pushRecentSearch returns the recent-search list with query moved to the front
// (deduped, capped). Session-only; not persisted.
func pushRecentSearch(recents []string, query string) []string {
	q := strings.TrimSpace(query)
	if q == "" {
		return recents
	}
	out := []string{q}
	for _, r := range recents {
		if r != q {
			out = append(out, r)
		}
	}
	if len(out) > maxRecentSearches {
		out = out[:maxRecentSearches]
	}
	return out
}

// openSearchView enters the search form, pre-filling the input with initial
// (empty for a fresh search, the last query when refining from results).
func (m model) openSearchView(initial string) (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "Search key, summary or description..."
	ti.CharLimit = 100
	ti.SetValue(initial)
	ti.Focus()
	m.searchInput = ti
	m.searchCursor = -1
	if m.mode != searchResultsView {
		m.previousMode = m.mode
	}
	m.mode = searchView
	return m, textinput.Blink
}

func (m model) updateSearchView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "esc":
			m.mode = m.previousMode
			return m, nil
		case "up":
			if m.searchCursor > -1 {
				m.searchCursor--
			}
			return m, nil
		case "down":
			if m.searchCursor < len(m.recentSearches)-1 {
				m.searchCursor++
			}
			return m, nil
		case "enter":
			query := m.searchInput.Value()
			if m.searchCursor >= 0 && m.searchCursor < len(m.recentSearches) {
				query = m.recentSearches[m.searchCursor]
			}
			query = strings.TrimSpace(query)
			if query == "" {
				return m, nil
			}
			m.recentSearches = pushRecentSearch(m.recentSearches, query)
			m.searchQuery = query
			m.searchResults = nil
			m.searchResultsCursor = 0
			m.mode = searchResultsView
			m.loadingCount++
			m.setInfo("Searching...")
			return m, m.searchIssuesCmd(query)
		}
	}

	prev := m.searchInput.Value()
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	if m.searchInput.Value() != prev {
		m.searchCursor = -1 // typing targets the typed query, not a recent
	}
	return m, cmd
}

func (m model) renderSearchView() string {
	var b strings.Builder
	b.WriteString(m.searchInput.View() + "\n")

	if len(m.recentSearches) > 0 {
		b.WriteString("\n" + ui.SectionTitleStyle.Render("Recent searches") + "\n")
		for i, r := range m.recentSearches {
			if i == m.searchCursor {
				b.WriteString(ui.IconCursor + ui.SelectedRowStyle.Render(" "+r) + "\n")
			} else {
				b.WriteString("  " + r + "\n")
			}
		}
	}

	return m.renderModal("Search Issues", b.String(), 0.4, 0.4)
}

func (m *model) refreshSearchResultsViewport() {
	width := m.windowWidth - ui.PanelOverheadWidth
	height := m.windowHeight - 6 // header block + footer + spacing
	if height < 3 {
		height = 3
	}
	m.searchResultsViewport.SetWidth(width)
	m.searchResultsViewport.SetHeight(height)

	content, cursorLine := m.buildSearchResultsContent()
	m.searchResultsViewport.SetContent(content)

	top := m.searchResultsViewport.YOffset()
	if cursorLine < top {
		m.searchResultsViewport.SetYOffset(cursorLine)
	} else if cursorLine >= top+height {
		m.searchResultsViewport.SetYOffset(cursorLine - height + 1)
	}
}

// buildSearchResultsContent renders the grouped result rows and returns the line
// index of the highlighted row so the viewport can keep it in view.
func (m model) buildSearchResultsContent() (string, int) {
	var b strings.Builder
	line := 0
	cursorLine := 0
	lastCat := searchCategory(-1)

	for i, r := range m.searchResults {
		if r.category != lastCat {
			b.WriteString(ui.SectionTitleStyle.Render(r.category.label()) + "\n")
			line++
			lastCat = r.category
		}
		if i == m.searchResultsCursor {
			cursorLine = line
		}
		b.WriteString(m.renderIssueRow(r.issue, i == m.searchResultsCursor, closureStatuses[r.issue.Status]) + "\n")
		line++
	}
	return b.String(), cursorLine
}

func (m model) updateSearchResultsView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "esc":
			return m.openSearchView(m.searchQuery)
		case "j", "down":
			if m.searchResultsCursor < len(m.searchResults)-1 {
				m.searchResultsCursor++
				m.refreshSearchResultsViewport()
			}
			return m, nil
		case "k", "up":
			if m.searchResultsCursor > 0 {
				m.searchResultsCursor--
				m.refreshSearchResultsViewport()
			}
			return m, nil
		case "g":
			m.searchResultsCursor = 0
			m.refreshSearchResultsViewport()
			return m, nil
		case "G":
			if len(m.searchResults) > 0 {
				m.searchResultsCursor = len(m.searchResults) - 1
				m.refreshSearchResultsViewport()
			}
			return m, nil
		case "enter":
			if m.searchResultsCursor >= 0 && m.searchResultsCursor < len(m.searchResults) {
				issue := m.searchResults[m.searchResultsCursor].issue
				m.selectedIssue = &issue
				m.detailReturnView = searchResultsView
				m.detailLayout = m.calculateDetailLayout()
				m.mode = detailView
				m.loadingCount++
				return m, m.fetchIssueDetailCmd(issue.Key)
			}
			return m, nil
		}
	}
	return m, nil
}

func (m model) renderSearchResultsView() string {
	header := fmt.Sprintf("Search: %q — %d result(s)", m.searchQuery, len(m.searchResults))
	var b strings.Builder
	b.WriteString(ui.DetailHeaderStyle.Render(header) + "\n\n")

	if len(m.searchResults) == 0 {
		b.WriteString(ui.StatusBarInfoStyle.Render("  No results") + "\n")
	} else {
		b.WriteString(m.renderListColumnsHeader() + "\n")
		b.WriteString(m.searchResultsViewport.View() + "\n")
	}

	footer := ui.StatusBarInfoStyle.Render("  ↑/↓ navigate · enter open · esc refine")
	return b.String() + "\n" + footer
}
