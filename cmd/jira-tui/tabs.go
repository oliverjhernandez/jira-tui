package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

const tabBarHeight = 1

const myIssuesJQL = "assignee = currentUser() AND resolution = Unresolved ORDER BY status DESC"

type tabKind int

const (
	tabMyIssues tabKind = iota
	tabEpicBoard
	tabSavedBoard
	tabProjectBoard
)

// boardState is the per-tab list/board state. sections and filteredSections are
// derived from issues on load, so they are not persisted here. selectedIssue is
// recomputed from the cursor on load (it aliases into sections and must never be
// stored).
type boardState struct {
	jql            string
	issues         []jira.Issue
	activeProjects []jira.Project
	filtering      bool
	filterValue    string
	cursor         int
	sectionCursor  int
	listYOffset    int
}

// detailState is the per-tab drill-down state. activeIssue is a self-contained
// fetched issue (its Comments/Worklogs/SubTasks are its own slices, not aliasing
// a board), so storing the pointer is safe. Viewports are not stored; only their
// scroll offsets, and they are rebuilt for the current window on load.
type detailState struct {
	activeIssue       *jira.Issue
	focusedSection    focusedSection
	commentsCursor    int
	worklogsCursor    int
	issueLinksCursor  int
	subTasksCursor    int
	descYOffset       int
	commentsYOffset   int
	worklogsYOffset   int
	issueLinksYOffset int
	subTasksYOffset   int
}

type Tab struct {
	id       int
	title    string
	kind     tabKind
	baseView viewMode // listView or detailView
	board    boardState
	detail   detailState
}

// SavedBoard is a predefined board available from the saved-board picker (B).
type SavedBoard struct {
	Title string
	JQL   string
}

// savedBoards is the v1 list of predefined boards. Edit in code to customize.
var savedBoards = []SavedBoard{
	{Title: "My Issues", JQL: myIssuesJQL},
	{Title: "Reported by me", JQL: "reporter = currentUser() AND resolution = Unresolved ORDER BY updated DESC"},
	{Title: "Updated recently", JQL: "assignee = currentUser() ORDER BY updated DESC"},
}

func (m model) activeTabID() int {
	if m.activeTab < 0 || m.activeTab >= len(m.tabs) {
		return 0
	}
	return m.tabs[m.activeTab].id
}

func (m model) activeBoardJQL() string {
	if m.activeTab < 0 || m.activeTab >= len(m.tabs) {
		return myIssuesJQL
	}
	return m.tabs[m.activeTab].board.jql
}

// tabIndexByID returns the index of the tab with the given id, or (-1, false).
func (m model) tabIndexByID(id int) (int, bool) {
	for i := range m.tabs {
		if m.tabs[i].id == id {
			return i, true
		}
	}
	return -1, false
}

// saveActiveTab snapshots the live (flat) fields into the active tab.
func (m *model) saveActiveTab() {
	if m.activeTab < 0 || m.activeTab >= len(m.tabs) {
		return
	}
	t := &m.tabs[m.activeTab]
	t.baseView = m.mode
	t.board = boardState{
		jql:            t.board.jql,
		issues:         m.issues,
		activeProjects: m.activeProjects,
		filtering:      m.filtering,
		filterValue:    m.textInput.Value(),
		cursor:         m.cursor,
		sectionCursor:  m.sectionCursor,
		listYOffset:    m.listViewport.YOffset(),
	}
	t.detail = detailState{
		activeIssue:       m.activeIssue,
		focusedSection:    m.focusedSection,
		commentsCursor:    m.commentsCursor,
		worklogsCursor:    m.worklogsCursor,
		issueLinksCursor:  m.IssueLinksCursor,
		subTasksCursor:    m.subTasksCursor,
		descYOffset:       m.descViewport.YOffset(),
		commentsYOffset:   m.commentsViewport.YOffset(),
		worklogsYOffset:   m.worklogsViewport.YOffset(),
		issueLinksYOffset: m.issueLinksViewport.YOffset(),
		subTasksYOffset:   m.subTasksViewport.YOffset(),
	}
}

// loadActiveTab restores the active tab into the live (flat) fields, recomputing
// layouts, sections and viewport contents for the current window. Returns a
// command to (re)start detail polling if the tab is on a detail view.
func (m *model) loadActiveTab() tea.Cmd {
	if m.activeTab < 0 || m.activeTab >= len(m.tabs) {
		return nil
	}
	t := m.tabs[m.activeTab]
	m.mode = t.baseView

	// --- board ---
	m.issues = t.board.issues
	m.activeProjects = t.board.activeProjects
	m.filtering = t.board.filtering
	m.textInput.SetValue(t.board.filterValue)
	if t.board.filtering {
		m.textInput.Focus()
	} else {
		m.textInput.Blur()
	}
	m.cursor = t.board.cursor
	m.sectionCursor = t.board.sectionCursor

	m.sections = m.classifyIssues(m.issues, m.statuses)
	if m.filtering && t.board.filterValue != "" {
		m.filteredSections = filterSections(m.sections, t.board.filterValue)
	} else {
		m.filteredSections = nil
	}

	m.listLayout = m.calculateListLayout()
	m.listViewport.SetWidth(m.listLayout.panelContentWidth)
	m.listViewport.SetHeight(m.listLayout.listHeight)
	m.listViewport.SetContent(m.buildListContent())
	m.listViewport.SetYOffset(t.board.listYOffset)

	if si, ok := m.currentIssue(); ok {
		m.selectedIssue = si
	} else {
		m.selectedIssue = nil
	}

	// --- detail ---
	m.activeIssue = t.detail.activeIssue
	m.focusedSection = t.detail.focusedSection
	m.commentsCursor = t.detail.commentsCursor
	m.worklogsCursor = t.detail.worklogsCursor
	m.IssueLinksCursor = t.detail.issueLinksCursor
	m.subTasksCursor = t.detail.subTasksCursor

	if m.activeIssue != nil {
		m.detailLayout = m.calculateDetailLayout()

		m.descViewport.SetWidth(m.detailLayout.leftColumnWidth)
		m.descViewport.SetHeight(m.detailLayout.descHeight)
		m.descViewport.SetContent(m.buildDescriptionContent(m.detailLayout.leftColumnWidth))
		m.descViewport.SetYOffset(t.detail.descYOffset)

		m.commentsViewport.SetWidth(m.detailLayout.leftColumnWidth)
		m.commentsViewport.SetHeight(m.detailLayout.commentsHeight)
		m.commentsViewport.SetContent(m.buildCommentsContent(m.detailLayout.leftColumnWidth))
		m.commentsViewport.SetYOffset(t.detail.commentsYOffset)

		m.worklogsViewport.SetWidth(m.detailLayout.rightColumnWidth)
		m.worklogsViewport.SetHeight(m.detailLayout.worklogsHeight)
		m.worklogsViewport.SetContent(m.buildWorklogsContent(m.detailLayout.rightColumnWidth - ui.PanelOverheadWidth))
		m.worklogsViewport.SetYOffset(t.detail.worklogsYOffset)

		m.issueLinksViewport.SetWidth(m.detailLayout.rightColumnWidth)
		m.issueLinksViewport.SetHeight(m.detailLayout.issueLinksHeight)
		m.issueLinksViewport.SetContent(m.buildIssueLinksContent(m.detailLayout.rightColumnWidth - ui.PanelOverheadWidth))
		m.issueLinksViewport.SetYOffset(t.detail.issueLinksYOffset)

		m.subTasksViewport.SetWidth(m.detailLayout.rightColumnWidth)
		m.subTasksViewport.SetHeight(m.detailLayout.subTasksHeight)
		m.subTasksViewport.SetContent(m.buildSubTasksContent(m.detailLayout.rightColumnWidth - ui.PanelOverheadWidth))
		m.subTasksViewport.SetYOffset(t.detail.subTasksYOffset)

		// Refresh the detail's worklogs and subtasks for the now-active tab
		// (also covers a detail that finished loading while backgrounded).
		m.loadingCount += 2
		return tea.Batch(
			m.fetchWorkLogsCmd(m.activeIssue.ID),
			m.fetchSubTasksCmd(m.activeIssue.Key),
		)
	}

	return nil
}

// activeProjectsFor returns the subset of projects that have issues in the set.
func activeProjectsFor(issues []jira.Issue, projects []jira.Project) []jira.Project {
	seen := make(map[string]bool)
	for _, i := range issues {
		seen[i.Project.ID] = true
	}
	var out []jira.Project
	for _, p := range projects {
		if seen[p.ID] {
			out = append(out, p)
		}
	}
	return out
}

func (m model) switchTab(dir int) (tea.Model, tea.Cmd) {
	if len(m.tabs) <= 1 {
		return m, nil
	}
	m.saveActiveTab()
	n := len(m.tabs)
	m.activeTab = ((m.activeTab+dir)%n + n) % n
	cmd := m.loadActiveTab()
	return m, cmd
}

func (m model) closeActiveTab() (tea.Model, tea.Cmd) {
	if len(m.tabs) <= 1 {
		m.setInfo("Can't close the last tab")
		return m, m.clearStatusAfter(clearMsgTimeout)
	}
	i := m.activeTab
	m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
	}
	cmd := m.loadActiveTab()
	return m, cmd
}

// openBoardTab opens (or focuses an existing) tab for the given JQL.
func (m model) openBoardTab(title, jql string, kind tabKind) (tea.Model, tea.Cmd) {
	for i := range m.tabs {
		if m.tabs[i].board.jql == jql {
			m.saveActiveTab()
			m.activeTab = i
			cmd := m.loadActiveTab()
			return m, cmd
		}
	}

	m.saveActiveTab()

	id := m.nextTabID
	m.nextTabID++
	m.tabs = append(m.tabs, Tab{
		id:       id,
		title:    title,
		kind:     kind,
		baseView: listView,
		board:    boardState{jql: jql},
	})
	m.activeTab = len(m.tabs) - 1

	// blank the live board so the new tab renders empty until issues arrive
	m.mode = listView
	m.issues = nil
	m.sections = nil
	m.filteredSections = nil
	m.activeProjects = nil
	m.filtering = false
	m.textInput.SetValue("")
	m.cursor = 0
	m.sectionCursor = 0
	m.selectedIssue = nil
	m.activeIssue = nil
	m.listViewport.SetContent("")

	m.loadingCount++
	return m, m.fetchBoardIssuesCmd(jql, id)
}

// deriveEpicBoard computes the title and JQL for a board scoped to the epic or
// parent of the given issue.
func deriveEpicBoard(src *jira.Issue) (title, jql string, err error) {
	if src == nil {
		return "", "", fmt.Errorf("no issue selected")
	}
	var key string
	if strings.EqualFold(src.Type, "Epic") {
		key = src.Key
	} else if src.Parent != nil && src.Parent.Key != "" {
		key = src.Parent.Key
	} else {
		return "", "", fmt.Errorf("no epic or parent for this issue")
	}
	return key, fmt.Sprintf("parent = %s ORDER BY status DESC", key), nil
}

func (m model) openEpicBoardTab() (tea.Model, tea.Cmd) {
	var src *jira.Issue
	if m.mode == detailView {
		src = m.activeIssue
	} else {
		src = m.selectedIssue
	}

	title, jql, err := deriveEpicBoard(src)
	if err != nil {
		m.setError("opening epic board", err)
		return m, m.clearStatusAfter(clearMsgTimeout)
	}

	return m.openBoardTab(title, jql, tabEpicBoard)
}

func (m model) renderTabBar() string {
	var b strings.Builder
	for i, t := range m.tabs {
		label := fmt.Sprintf(" %d:%s ", i+1, t.title)
		if i == m.activeTab {
			b.WriteString(ui.TabActiveStyle.Render(label))
		} else {
			b.WriteString(ui.TabInactiveStyle.Render(label))
		}
		if i < len(m.tabs)-1 {
			b.WriteString(ui.TabBarStyle.Render("│"))
		}
	}
	return ui.TabBarStyle.
		Width(m.windowWidth).
		MaxWidth(m.windowWidth).
		MaxHeight(1).
		Render(b.String())
}
