package main

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

const maxRecentSearches = 10

// Modal size for the search view (fraction of the terminal).
const (
	searchModalWScale = 0.8
	searchModalHScale = 0.7
)

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

// openSearchView enters a fresh search modal (empty input, showing recents).
func (m model) openSearchView() (tea.Model, tea.Cmd) {
	ti := textinput.New()
	ti.Placeholder = "Search key, summary or description..."
	ti.CharLimit = 100
	ti.Focus()
	m.searchInput = ti
	m.searchCursor = -1
	m.searched = false
	m.searchResults = nil
	m.searchQuery = ""
	m.previousMode = m.mode
	m.mode = searchView
	return m, textinput.Blink
}

// searchListLen is the number of navigable rows below the input: results after a
// search has run, otherwise recent searches.
func (m model) searchListLen() int {
	if m.searched {
		return len(m.searchResults)
	}
	return len(m.recentSearches)
}

func (m model) updateSearchView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "esc":
			m.mode = m.previousMode
			return m, nil
		case "up", "ctrl+k":
			if m.searchCursor > -1 {
				m.searchCursor--
				m.refreshSearchResultsViewport()
			}
			return m, nil
		case "down", "ctrl+j":
			if m.searchCursor < m.searchListLen()-1 {
				m.searchCursor++
				m.refreshSearchResultsViewport()
			}
			return m, nil
		case "enter":
			return m.submitSearch()
		}
	}

	prev := m.searchInput.Value()
	var cmd tea.Cmd
	m.searchInput, cmd = m.searchInput.Update(msg)
	if m.searchInput.Value() != prev {
		m.searchCursor = -1 // typing targets the typed query, not a list row
	}
	return m, cmd
}

// submitSearch handles Enter: open a highlighted result, re-run a highlighted
// recent search, or run the typed query.
func (m model) submitSearch() (tea.Model, tea.Cmd) {
	if m.searchCursor >= 0 {
		if m.searched {
			if m.searchCursor < len(m.searchResults) {
				issue := m.searchResults[m.searchCursor].issue
				m.selectedIssue = &issue
				m.detailReturnView = searchView
				m.detailLayout = m.calculateDetailLayout()
				m.mode = detailView
				m.loadingCount++
				return m, m.fetchIssueDetailCmd(issue.Key)
			}
			return m, nil
		}
		if m.searchCursor < len(m.recentSearches) {
			return m.runSearch(m.recentSearches[m.searchCursor])
		}
	}
	return m.runSearch(m.searchInput.Value())
}

func (m model) runSearch(query string) (tea.Model, tea.Cmd) {
	query = strings.TrimSpace(query)
	if query == "" {
		return m, nil
	}
	m.recentSearches = pushRecentSearch(m.recentSearches, query)
	m.searchInput.SetValue(query)
	m.searchQuery = query
	m.searchResults = nil
	m.searchCursor = -1
	m.loadingCount++
	m.setInfo("Searching...")
	return m, m.searchIssuesCmd(query)
}

// searchModalInnerSize returns the width/height available for the results
// viewport inside the modal box.
func (m model) searchModalInnerSize() (int, int) {
	w := ui.GetModalWidth(m.windowWidth, searchModalWScale) - ui.PanelOverheadWidth
	h := ui.GetModalHeight(m.windowHeight, searchModalHScale) - ui.PanelOverheadHeight - 4
	if w < 20 {
		w = 20
	}
	if h < 3 {
		h = 3
	}
	return w, h
}

func (m *model) refreshSearchResultsViewport() {
	w, h := m.searchModalInnerSize()
	m.searchResultsViewport.SetWidth(w)
	m.searchResultsViewport.SetHeight(h)

	content, cursorLine := m.buildSearchResultsContent(w)
	m.searchResultsViewport.SetContent(content)

	top := m.searchResultsViewport.YOffset()
	if cursorLine < top {
		m.searchResultsViewport.SetYOffset(cursorLine)
	} else if cursorLine >= top+h {
		m.searchResultsViewport.SetYOffset(cursorLine - h + 1)
	}
}

// buildSearchResultsContent renders compact, modal-width result rows grouped by
// match category, and returns the line index of the highlighted row.
func (m model) buildSearchResultsContent(width int) (string, int) {
	var b strings.Builder
	line, cursorLine := 0, 0
	lastCat := searchCategory(-1)

	for i, r := range m.searchResults {
		if r.category != lastCat {
			b.WriteString(ui.SectionTitleStyle.Render(r.category.label()) + "\n")
			line++
			lastCat = r.category
		}

		is := r.issue
		typeIcon := ui.RenderIssueType(is.Type, false)
		key := ui.PadCell(is.Key, 12)
		status := ui.RenderStatusBadge(is.Status)
		used := 2 + lipgloss.Width(typeIcon) + 1 + 12 + 1 + lipgloss.Width(status) + 1
		summaryW := width - used
		if summaryW < 10 {
			summaryW = 10
		}
		summary := ui.PadCell(is.Summary, summaryW)
		row := typeIcon + " " + key + " " + status + " " + summary

		if i == m.searchCursor {
			cursorLine = line
			b.WriteString(ui.IconCursor + ui.SelectedRowStyle.Render(row) + "\n")
		} else {
			b.WriteString("  " + ui.NormalRowStyle.Render(row) + "\n")
		}
		line++
	}
	return b.String(), cursorLine
}

func (m model) renderSearchView() string {
	var b strings.Builder
	b.WriteString(m.searchInput.View() + "\n\n")

	switch {
	case m.searched && len(m.searchResults) == 0:
		b.WriteString(ui.StatusBarInfoStyle.Render(fmt.Sprintf("No results for %q", m.searchQuery)))
	case m.searched:
		b.WriteString(m.searchResultsViewport.View())
	case len(m.recentSearches) > 0:
		b.WriteString(ui.SectionTitleStyle.Render("Recent searches") + "\n")
		for i, r := range m.recentSearches {
			if i == m.searchCursor {
				b.WriteString(ui.IconCursor + ui.SelectedRowStyle.Render(" "+r) + "\n")
			} else {
				b.WriteString("  " + r + "\n")
			}
		}
	default:
		b.WriteString(ui.StatusBarInfoStyle.Render("Type a query and press enter"))
	}

	b.WriteString("\n" + ui.StatusBarInfoStyle.Render("↑/↓ select · enter open/search · esc close"))
	return m.renderModal("Search Issues", b.String(), searchModalWScale, searchModalHScale)
}
