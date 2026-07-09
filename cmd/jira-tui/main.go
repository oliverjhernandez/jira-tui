// cmd/jira-tui/main.go
package main

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"time"

	tea "charm.land/bubbletea/v2"

	"charm.land/bubbles/v2/spinner"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"

	"github.com/oliverjhernandez/jira-tui/internal/config"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

const jiraURL = "https://layer7.atlassian.net/browse/"

const clearMsgTimeout = 5 * time.Second

type viewMode int

const (
	listView viewMode = iota
	newIssueView
	detailView
	transitionView
	userSearchView
	descriptionView
	priorityView
	commentView
	worklogView
	issueLinkView
	estimateView
	cancelReasonView
	blockReasonView
	issueSearchView
	savedBoardPickerView
)

func (v viewMode) String() string {
	switch v {
	case listView:
		return "listView"
	case newIssueView:
		return "newIssueView"
	case detailView:
		return "detailView"
	case transitionView:
		return "transitionView"
	case userSearchView:
		return "userSearchView"
	case descriptionView:
		return "descriptionView"
	case priorityView:
		return "priorityView"
	case commentView:
		return "commentView"
	case worklogView:
		return "worklogView"
	case issueLinkView:
		return "issueLinkView"
	case estimateView:
		return "estimateView"
	case cancelReasonView:
		return "cancelReasonView"
	case blockReasonView:
		return "blockReasonView"
	case issueSearchView:
		return "issueSearchView"
	case savedBoardPickerView:
		return "savedBoardPickerView"
	default:
		return "unknown"
	}
}

// isModal reports whether a view is drawn as an overlay on top of a base
// (full-screen) view rather than being a base view itself.
func (v viewMode) isModal() bool {
	switch v {
	case listView, detailView:
		return false
	default:
		return true
	}
}

type userSelectionMode int

const (
	assignUser userSelectionMode = iota
	insertMention
)

type issueSelectionMode int

const (
	standardIssueSearch issueSelectionMode = iota
	linkIssue
)

type focusedSection int

const (
	metadataSection focusedSection = iota
	descriptionSection
	commentsSection
	worklogsSection
	issueLinksSection
	subTasksSection
)

func (f focusedSection) String() string {
	switch f {
	case descriptionSection:
		return "descSection"
	case commentsSection:
		return "commentsSection"
	case worklogsSection:
		return "worklogsSection"
	case subTasksSection:
		return "subTasksSection"
	default:
		return "unknown"
	}
}

type Section struct {
	Name        string
	CategoryKey string
	Collapsed   bool
	Issues      []jira.Issue
}

type model struct {
	// Core
	client       *jira.Client
	mode         viewMode
	previousMode viewMode
	baseView     viewMode
	err          error

	// Tabs
	tabs      []Tab
	activeTab int
	nextTabID int

	windowWidth int
	// Window & Layout
	windowHeight       int
	detailLayout       detailLayout
	listLayout         listLayout
	columnWidths       ui.ColumnWidths
	listViewport       viewport.Model
	descViewport       viewport.Model
	commentsViewport   viewport.Model
	worklogsViewport   viewport.Model
	issueLinksViewport viewport.Model
	subTasksViewport   viewport.Model

	// User Data
	myself *jira.User

	// Issue Data
	issues         []jira.Issue
	activeIssue    *jira.Issue
	pendingIssue   *jira.Issue
	projects       []jira.Project
	activeProjects []jira.Project
	issueTypes     []jira.IssueType
	selectedIssue  *jira.Issue

	// Issue Metadata
	sections         []Section
	focusedSection   focusedSection
	filteredSections []Section
	statuses         map[string][]jira.Status
	priorities       []jira.Priority

	// Worklogs
	worklogTotals map[string]int

	// Transitions
	// transitions       map[string][]jira.Transition
	pendingTransition *jira.Transition
	transitionCache   map[string]map[string][]jira.Transition

	//  Selection
	usersCache         []jira.User
	userSelectionMode  userSelectionMode
	issueSelectionMode issueSelectionMode

	// Navigation & Cursors
	cursor           int
	sectionCursor    int
	transitionCursor int
	userCursor       int
	commentsCursor   int
	worklogsCursor   int
	IssueLinksCursor int
	subTasksCursor   int

	// Input Components
	textInput textinput.Model
	textArea  textarea.Model
	filtering bool
	lastKey   string

	// Editing State
	editingDescription bool
	editingComment     bool
	editingWorklog     bool

	// Form Data
	worklogFormData  *WorklogFormData
	newIssueData     *NewIssueFormData
	estimateData     *EstimateFormData
	searchIssueData  *SearchIssueFormData
	issueLinkData    *IssueLinkFormData
	commentData      *CommentFormData
	descriptionData  *DescriptionFormData
	priorityData     *PriorityFormData
	transitionData   *TransitionFormData
	cancelReasonData *CancelReasonFormData
	blockReasonData  *BlockReasonFormData
	searchUserData   *SearchUserFormData
	savedBoardData   *SavedBoardFormData

	// UI Elements
	spinner       spinner.Model
	statusMessage statusMessage
	loadingCount  int

	// Pollers
	detailPolling bool
}

func (m model) Init() tea.Cmd {

	var cmds []tea.Cmd

	cmds = append(cmds, tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return myIssuesPollMsg{}
	}))

	// Single perpetual detail-poll chain: it refreshes whatever detail is active
	// (see issueDetailPollMsg). Started once here so exactly one chain exists.
	cmds = append(cmds, tea.Tick(time.Minute, func(t time.Time) tea.Msg {
		return issueDetailPollMsg{}
	}))

	cmds = append(cmds, m.spinner.Tick)
	cmds = append(cmds, m.fetchMySelfCmd())
	cmds = append(cmds, m.fetchProjectsCmd())
	cmds = append(cmds, m.fetchMyIssuesCmd())
	cmds = append(cmds, m.fetchPrioritiesCmd())
	cmds = append(cmds, m.fetchAllUsersCmd())
	cmds = append(cmds, m.fetchIssueTypesCmd())

	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	next, cmd := m.update(msg)
	nm := next.(model)
	// Record the last full-screen view so modal overlays always composite over
	// whatever base view the user was on, without each modal inferring it.
	if !nm.mode.isModal() {
		nm.baseView = nm.mode
	}
	return nm, cmd
}

func (m model) update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Tab management keys, intercepted globally in base views (not in modals or
	// while filtering, where these characters are legitimate input).
	if keyMsg, ok := msg.(tea.KeyPressMsg); ok && !m.mode.isModal() && !m.filtering {
		switch keyMsg.String() {
		case "]":
			return m.switchTab(+1)
		case "[":
			return m.switchTab(-1)
		case "b":
			return m.openEpicBoardTab()
		case "x":
			return m.closeActiveTab()
		case "B":
			return m.openSavedBoardPicker()
		}
	}

	switch msg := msg.(type) {
	case myselfLoadedMsg:
		m.loadingCount--
		m.myself = msg.me
		return m, nil

	case issuesLoadedMsg:
		m.loadingCount--
		idx, ok := m.tabIndexByID(msg.tabID)
		if !ok {
			return m, nil // tab was closed; drop
		}

		aps := activeProjectsFor(msg.issues, m.projects)
		if idx == m.activeTab {
			m.issues = msg.issues
			m.activeProjects = aps
		} else {
			m.tabs[idx].board.issues = msg.issues
			m.tabs[idx].board.activeProjects = aps
		}

		var cmds []tea.Cmd
		if len(m.projects) > 0 {
			m.loadingCount++
			cmds = append(cmds, m.fetchStatusesCmd(aps, msg.tabID))

			m.loadingCount++
			cmds = append(cmds, m.fetchAllWorklogsTotalCmd(msg.issues, msg.tabID))
		}

		return m, tea.Batch(cmds...)

	case subTasksLoadedMsg:
		m.loadingCount--
		idx, ok := m.tabIndexByID(msg.tabID)
		if !ok {
			return m, nil
		}
		if idx == m.activeTab {
			m.searchIssueData = NewSearchIssueFormData()
			if m.activeIssue != nil {
				m.activeIssue.SubTasks = msg.subTasks

				subTasksContent := m.buildSubTasksContent(m.detailLayout.rightColumnWidth - ui.PanelOverheadWidth)
				m.subTasksViewport.SetWidth(m.detailLayout.rightColumnWidth)
				m.subTasksViewport.SetHeight(m.detailLayout.subTasksHeight)
				m.subTasksViewport.SetContent(subTasksContent)
			}
		} else if m.tabs[idx].detail.activeIssue != nil {
			m.tabs[idx].detail.activeIssue.SubTasks = msg.subTasks
		}

		return m, nil

	case worklogTotalsLoadedMsg:
		m.loadingCount--
		if m.worklogTotals == nil {
			m.worklogTotals = make(map[string]int)
		}
		maps.Copy(m.worklogTotals, msg.totals) // shared cache, always merge
		if idx, ok := m.tabIndexByID(msg.tabID); ok && idx == m.activeTab {
			m.listViewport.SetContent(m.buildListContent())
		}
		return m, nil

	case prioritiesLoadedMsg:
		m.loadingCount--
		m.priorities = msg.priorities
		return m, nil

	case projectsLoadedMsg:
		m.loadingCount--
		m.projects = msg.projects
		var cmds []tea.Cmd

		if len(m.projects) > 0 && len(m.issues) > 0 {
			m.activeProjects = activeProjectsFor(m.issues, m.projects)
			m.loadingCount++
			cmds = append(cmds, m.fetchStatusesCmd(m.activeProjects, m.activeTabID()))
		}

		return m, tea.Batch(cmds...)

	case issueDetailLoadedMsg:
		m.loadingCount--
		idx, ok := m.tabIndexByID(msg.tabID)
		if !ok {
			return m, nil
		}

		if idx != m.activeTab {
			// Detail finished loading for a backgrounded tab: stash it. Its
			// worklogs/subtasks are fetched when the user switches to it
			// (loadActiveTab).
			m.tabs[idx].detail.activeIssue = msg.detail
			m.tabs[idx].baseView = detailView
			return m, nil
		}

		var cmds []tea.Cmd
		m.activeIssue = msg.detail
		m.detailLayout = m.calculateDetailLayout()
		m.previousMode = m.mode
		m.mode = detailView

		if m.activeIssue != nil {
			m.commentsViewport.SetWidth(m.detailLayout.leftColumnWidth)
			descContent := m.buildDescriptionContent(m.detailLayout.leftColumnWidth)
			m.descViewport.SetHeight(m.detailLayout.descHeight)
			m.descViewport.SetWidth(m.detailLayout.leftColumnWidth)
			m.descViewport.SetContent(descContent)

			commentsContent := m.buildCommentsContent(m.detailLayout.leftColumnWidth)
			m.commentsViewport.SetHeight(m.detailLayout.commentsHeight)
			m.commentsViewport.SetWidth(m.detailLayout.leftColumnWidth)
			m.commentsViewport.SetContent(commentsContent)
		}

		issueLinksContent := m.buildIssueLinksContent(m.detailLayout.rightColumnWidth - ui.PanelOverheadWidth)
		m.issueLinksViewport.SetWidth(m.detailLayout.rightColumnWidth)
		m.issueLinksViewport.SetHeight(m.detailLayout.issueLinksHeight)
		m.issueLinksViewport.SetContent(issueLinksContent)

		m.loadingCount++
		worklogsCmd := m.fetchWorkLogsCmd(m.activeIssue.ID)
		cmds = append(cmds, worklogsCmd)

		m.loadingCount++
		subTasksCmd := m.fetchSubTasksCmd(m.activeIssue.Key)
		cmds = append(cmds, subTasksCmd)

		return m, tea.Batch(cmds...)

	case workLogsLoadedMsg:
		m.loadingCount--
		idx, ok := m.tabIndexByID(msg.tabID)
		if !ok {
			return m, nil
		}

		var total int
		for _, wl := range msg.workLogs {
			total += wl.Time
		}
		if m.worklogTotals == nil {
			m.worklogTotals = make(map[string]int)
		}

		if idx == m.activeTab {
			if m.activeIssue != nil {
				m.activeIssue.Worklogs = msg.workLogs
				m.worklogTotals[m.activeIssue.ID] = total

				worklogsContent := m.buildWorklogsContent(m.detailLayout.rightColumnWidth - ui.PanelOverheadWidth)
				m.worklogsViewport.SetWidth(m.detailLayout.rightColumnWidth)
				m.worklogsViewport.SetHeight(m.detailLayout.worklogsHeight)
				m.worklogsViewport.SetContent(worklogsContent)
			}
		} else if m.tabs[idx].detail.activeIssue != nil {
			m.tabs[idx].detail.activeIssue.Worklogs = msg.workLogs
			m.worklogTotals[m.tabs[idx].detail.activeIssue.ID] = total
		}

		return m, nil

	case issueTypesLoadedMsg:
		m.loadingCount--
		m.issueTypes = msg.issueTypes
		return m, nil

	case transitionsLoadedMsg:
		m.loadingCount--
		if m.transitionCache[msg.projectKey] == nil {
			m.transitionCache[msg.projectKey] = make(map[string][]jira.Transition)
		}
		m.transitionCache[msg.projectKey][msg.status] = msg.transitions
		if m.mode == transitionView {
			m.transitionData = NewTransitionFormData(msg.transitions)
			return m, m.transitionData.Form.Init()
		}
		return m, nil

	case statusesLoadedMsg:
		m.loadingCount--
		if m.statuses == nil {
			m.statuses = make(map[string][]jira.Status)
		}
		maps.Copy(m.statuses, msg.statuses) // shared cache across boards

		idx, ok := m.tabIndexByID(msg.tabID)
		if !ok || idx != m.activeTab {
			// Inactive tab: classification is deferred to loadActiveTab.
			return m, nil
		}

		m.sections = m.classifyIssues(m.issues, m.statuses)
		m.listViewport.SetContent(m.buildListContent())

		m.selectedIssue = nil
		for si := range m.sections {
			if len(m.sections[si].Issues) > 0 {
				m.sectionCursor = si
				m.cursor = 0
				m.selectedIssue = &m.sections[si].Issues[0]
				break
			}
		}

		return m, nil

	case transitionPostedMsg:
		m.loadingCount--
		m.setSuccess("Issue transitioned")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		switch m.focusedSection {
		case metadataSection:
			m.loadingCount++
			cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		case subTasksSection:
			m.loadingCount++
			cmds = append(cmds, m.fetchSubTasksCmd(m.activeIssue.Key))
		}

		return m, tea.Batch(cmds...)

	case assigneePostedMsg:
		m.loadingCount--
		m.setSuccess("User assigned successfully")
		return m, nil

	case newIssuePostedMsg:
		m.loadingCount--
		var cmds []tea.Cmd
		if m.activeIssue != nil {
			m.mode = detailView
			m.loadingCount++
			cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		} else {
			m.mode = listView
			m.loadingCount++
			cmds = append(cmds, m.fetchMyIssuesCmd())
		}
		m.setSuccess("New issue created successfully")
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		return m, tea.Batch(cmds...)

	case issueLinkPostedMsg:
		m.loadingCount--
		m.setSuccess("Issue linked successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		return m, tea.Batch(cmds...)

	case updatedDescriptionMsg:
		m.loadingCount--
		var cmds []tea.Cmd
		m.setSuccess("Description edited successfully")
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		return m, tea.Batch(cmds...)

	case priorityPostedMsg:
		m.loadingCount--
		m.setSuccess("Priority posted successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		return m, tea.Batch(cmds...)

	case commentPostedMsg:
		m.loadingCount--
		m.setSuccess("Comment posted successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		return m, tea.Batch(cmds...)

	case commentUpdatedMsg:
		m.loadingCount--
		m.setSuccess("Comment edited successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		return m, tea.Batch(cmds...)

	case commentDeletedMsg:
		m.loadingCount--
		m.setSuccess("Comment deleted successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		return m, tea.Batch(cmds...)

	case workLogPostedMsg:
		m.loadingCount--
		m.setSuccess("Worklog posted successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		m.loadingCount++
		cmds = append(cmds, m.fetchWorkLogsCmd(m.activeIssue.ID))
		return m, tea.Batch(cmds...)

	case workLogUpdatedMsg:
		m.loadingCount--
		m.setSuccess("Worklog edited successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		m.loadingCount++
		cmds = append(cmds, m.fetchWorkLogsCmd(m.activeIssue.ID))
		return m, tea.Batch(cmds...)

	case workLogDeletedMsg:
		m.loadingCount--
		m.setSuccess("Worklog deleted successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		m.loadingCount++
		cmds = append(cmds, m.fetchWorkLogsCmd(m.activeIssue.ID))
		return m, tea.Batch(cmds...)

	case estimatePostedMsg:
		m.loadingCount--
		m.setSuccess("Estimate posted successfully")
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		if m.pendingTransition != nil {
			transition := m.pendingTransition
			if isCancelTransition(*transition) {
				m.cancelReasonData = NewCancelReasonFormData()
				m.mode = cancelReasonView
				return m, m.cancelReasonData.Form.Init()
			}
			if isBlockedTransition(*transition) {
				m.blockReasonData = NewBlockReasonFormData()
				m.mode = blockReasonView
				return m, m.blockReasonData.Form.Init()
			}
			m.pendingTransition = nil
			cmds = append(cmds, m.postTransitionCmd(m.activeIssue.Key, transition.ID, transition.Name))
			return m, tea.Batch(cmds...)
		}
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
		return m, tea.Batch(cmds...)

	case usersLoadedMsg:
		m.loadingCount--
		m.usersCache = msg.users
		return m, nil

	case myIssuesPollMsg:
		var cmds []tea.Cmd
		m.loadingCount++
		cmds = append(cmds, m.fetchMyIssuesCmd())
		cmds = append(cmds, tea.Tick(time.Minute, func(t time.Time) tea.Msg {
			return myIssuesPollMsg{}
		}))
		m.setInfo("Fetching my issues...")
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		return m, tea.Batch(cmds...)

	case issueDetailPollMsg:
		var cmds []tea.Cmd

		// Perpetual single chain: always reschedule, refresh whichever detail
		// is currently active.
		cmds = append(cmds, tea.Tick(time.Minute, func(t time.Time) tea.Msg {
			return issueDetailPollMsg{}
		}))

		if m.mode == detailView && m.activeIssue != nil {
			m.loadingCount++
			cmds = append(cmds, m.fetchIssueDetailCmd(m.activeIssue.Key))
			m.setInfo("Fetching issue details...")
			cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		}

		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		var tickCmd tea.Cmd
		m.spinner, tickCmd = m.spinner.Update(msg)
		return m, tickCmd

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width

		if m.activeIssue != nil {
			m.detailLayout = m.calculateDetailLayout()

			descContent := m.buildDescriptionContent(m.detailLayout.leftColumnWidth)
			m.descViewport.SetWidth(m.detailLayout.leftColumnWidth)
			m.descViewport.SetHeight(m.detailLayout.descHeight)
			m.descViewport.SetContent(descContent)

			commentsContent := m.buildCommentsContent(m.detailLayout.leftColumnWidth)
			m.commentsViewport.SetWidth(m.detailLayout.leftColumnWidth)
			m.commentsViewport.SetHeight(m.detailLayout.commentsHeight)
			m.commentsViewport.SetContent(commentsContent)
		}

		m.listLayout = m.calculateListLayout()
		m.listViewport.SetWidth(m.listLayout.panelContentWidth)
		m.listViewport.SetHeight(m.listLayout.listHeight)

		m.columnWidths = ui.CalculateColumnWidths(msg.Width)
		m.listViewport.SetContent(m.buildListContent())

		return m, nil

	case keyTimeoutMsg:
		m.lastKey = ""
		return m, nil

	case clearStatusMsg:
		m.statusMessage.content = ""
		return m, nil

	case errMsg:
		var cmds []tea.Cmd

		m.loadingCount--
		m.setError("request failed", msg.err)
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		if m.mode == issueSearchView && m.searchIssueData != nil {
			m.searchIssueData = NewSearchIssueFormData()
			m.searchIssueData.Err = msg.err
		}

		return m, tea.Batch(cmds...)
	}

	var viewCmd tea.Cmd
	var tmpModel tea.Model
	switch m.mode {
	case listView:
		tmpModel, viewCmd = m.updateListView(msg)
	case newIssueView:
		tmpModel, viewCmd = m.updateNewIssueView(msg)
	case detailView:
		tmpModel, viewCmd = m.updateDetailView(msg)
	case descriptionView:
		tmpModel, viewCmd = m.updateEditDescriptionView(msg)
	case priorityView:
		tmpModel, viewCmd = m.updateEditPriorityView(msg)
	case transitionView:
		tmpModel, viewCmd = m.updateTransitionView(msg)
	case commentView:
		tmpModel, viewCmd = m.updateCommentView(msg)
	case userSearchView:
		tmpModel, viewCmd = m.updateSearchUserView(msg)
	case worklogView:
		tmpModel, viewCmd = m.updateWorklogView(msg)
	case issueLinkView:
		tmpModel, viewCmd = m.updateIssueLinkView(msg)
	case estimateView:
		tmpModel, viewCmd = m.updatePostEstimateView(msg)
	case cancelReasonView:
		tmpModel, viewCmd = m.updatePostCancelReasonView(msg)
	case blockReasonView:
		tmpModel, viewCmd = m.updateBlockReasonView(msg)
	case issueSearchView:
		tmpModel, viewCmd = m.updateSearchIssueView(msg)
	case savedBoardPickerView:
		tmpModel, viewCmd = m.updateSavedBoardPickerView(msg)
	}

	m = tmpModel.(model)
	return m, tea.Batch(viewCmd)
}

func (m model) View() tea.View {
	var content string

	if m.err != nil {
		return tea.NewView(fmt.Sprintf("Error: %v\n\nPress 'q' to quit.\n", m.err))
	}

	switch m.mode {
	case listView:
		content = m.renderListView()
	case newIssueView:
		content = m.renderNewIssueView()
	case detailView:
		content = m.renderDetailView()
	case transitionView:
		content = m.renderTransitionView()
	case descriptionView:
		content = m.renderEditDescriptionView()
	case priorityView:
		content = m.renderEditPriorityView()
	case commentView:
		content = m.renderCommentView()
	case userSearchView:
		content = m.renderSearchUserView()
	case worklogView:
		content = m.renderWorklogView()
	case issueLinkView:
		content = m.renderIssueLinkView()
	case estimateView:
		content = m.renderPostEstimateView()
	case cancelReasonView:
		content = m.renderPostCancelReasonView()
	case blockReasonView:
		content = m.renderBlockReasonView()
	case issueSearchView:
		content = m.renderSearchIssueView()
	case savedBoardPickerView:
		content = m.renderSavedBoardPickerView()
	default:
		content = "Unknown view\n"
	}

	return tea.NewView(content)
}

// Build metadata, overridden via -ldflags at release time (see .goreleaser.yaml).
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "--version", "-v", "version":
			fmt.Printf("jira-tui %s (%s, %s)\n", version, commit, date)
			return
		}
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		panic(err)
	}

	client, _ := jira.NewClient(cfg.JiraURL, cfg.JIraEmail, cfg.JiraToken, cfg.TempoURL, cfg.TempoToken)

	logFile, err := os.OpenFile("debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := logFile.Close(); err != nil {
			fmt.Printf("error: %s", err)
		}
	}()
	slog.SetDefault(slog.New(slog.NewTextHandler(logFile, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	})))

	textInput := textinput.New()
	textInput.CharLimit = 50

	spinner := spinner.New()

	p := tea.NewProgram(model{
		mode:            listView,
		baseView:        listView,
		client:          client,
		textInput:       textInput,
		windowWidth:     80,
		windowHeight:    24,
		spinner:         spinner,
		worklogTotals:   make(map[string]int),
		columnWidths:    ui.CalculateColumnWidths(80),
		loadingCount:    6, // Init cmds
		transitionCache: make(map[string]map[string][]jira.Transition, 0),
		activeTab:       0,
		nextTabID:       1,
		tabs: []Tab{{
			id:       0,
			title:    "My Issues",
			kind:     tabMyIssues,
			baseView: listView,
			board:    boardState{jql: myIssuesJQL},
		}},
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
