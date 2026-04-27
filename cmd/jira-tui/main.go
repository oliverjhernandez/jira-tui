// cmd/jira-tui/main.go
package main

import (
	"fmt"
	"log"
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
	issueSearchView
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
	case issueSearchView:
		return "issueSearchView"
	default:
		return "unknown"
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

type messageType int

const (
	infoStatusBarMsg = iota
	errStatusBarMsg
)

type statusMessage struct {
	content string
	msgType messageType
}

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
	Issues      []*jira.Issue
}

type model struct {
	// Core
	client       *jira.Client
	mode         viewMode
	previousMode viewMode
	err          error

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
	projects       []jira.Project
	activeProjects []jira.Project
	issueTypes     []jira.IssueType
	selectedIssue  *jira.Issue
	issueDetail    *jira.IssueDetail

	// Issue Metadata
	sections         []Section
	focusedSection   focusedSection
	filteredSections []Section
	statuses         []jira.Status
	priorities       []jira.Priority

	// Worklogs
	worklogTotals map[string]int

	// Transitions
	transitions       []jira.Transition
	pendingTransition *jira.Transition
	activeIssue       *jira.Issue

	//  Selection
	usersCache         []jira.User
	filteredUsers      []*jira.User
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
	// TODO: Move to huh form
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
	searchData       *SearchIssueFormData
	issueLinkData    *IssueLinkFormData
	commentData      *CommentFormData
	descriptionData  *DescriptionFormData
	priorityData     *PriorityFormData
	transitionData   *TransitionFormData
	cancelReasonData *CancelReasonFormData

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
	switch msg := msg.(type) {
	case myselfLoadedMsg:
		m.loadingCount--
		m.myself = msg.me
		return m, nil

	case issuesLoadedMsg:
		m.loadingCount--
		m.issues = msg.issues

		var cmds []tea.Cmd

		if len(m.projects) > 0 && len(m.issues) > 0 {
			seen := make(map[string]bool, 0)
			for _, p := range m.issues {
				seen[p.Project.ID] = true
			}

			m.activeProjects = nil
			for _, p := range m.projects {
				if seen[p.ID] {
					m.activeProjects = append(m.activeProjects, p)
				}
			}

			m.loadingCount++
			cmds = append(cmds, m.fetchStatusesCmd(m.activeProjects))

			m.loadingCount++
			cmds = append(cmds, m.fetchAllWorklogsTotalCmd(msg.issues))
		}

		return m, tea.Batch(cmds...)

	case subTasksLoadedMsg:
		m.loadingCount--
		m.searchData = NewSearchFormData()
		if m.issueDetail != nil {
			m.issueDetail.SubTasks = msg.subTasks
		}

		subTasksContent := m.buildSubTasksContent(m.detailLayout.rightColumnWidth - ui.PanelOverheadWidth)
		m.subTasksViewport.SetWidth(m.detailLayout.rightColumnWidth)
		m.subTasksViewport.SetHeight(m.detailLayout.subTasksHeight)
		m.subTasksViewport.SetContent(subTasksContent)

		return m, nil

	case worklogTotalsLoadedMsg:
		m.loadingCount--
		if m.worklogTotals == nil {
			m.worklogTotals = make(map[string]int)
		}
		maps.Copy(m.worklogTotals, msg.totals)
		m.listViewport.SetContent(m.buildListContent())
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
			seen := make(map[string]bool, 0)
			for _, p := range m.issues {
				seen[p.Project.ID] = true
			}

			for _, p := range m.projects {
				if seen[p.ID] {
					m.activeProjects = append(m.activeProjects, p)
				}
			}

			m.loadingCount++
			cmds = append(cmds, m.fetchStatusesCmd(m.activeProjects))
		}

		return m, tea.Batch(cmds...)

	case issueDetailLoadedMsg:
		var cmds []tea.Cmd
		m.loadingCount--
		m.issueDetail = msg.detail
		m.detailLayout = m.calculateDetailLayout()
		m.previousMode = m.mode
		m.mode = detailView

		if !m.detailPolling {
			m.detailPolling = true
			cmds = append(cmds, tea.Tick(time.Minute, func(t time.Time) tea.Msg {
				return issueDetailPollMsg{}
			}))
		}

		if m.issueDetail != nil {
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
		worklogsCmd := m.fetchWorkLogsCmd(m.issueDetail.ID)
		cmds = append(cmds, worklogsCmd)

		m.loadingCount++
		subTasksCmd := m.fetchSubTasksCmd(m.issueDetail.Key)
		cmds = append(cmds, subTasksCmd)

		return m, tea.Batch(cmds...)

	case workLogsLoadedMsg:
		m.loadingCount--
		if m.issueDetail != nil {
			m.issueDetail.Worklogs = msg.workLogs
			var total int
			for _, wl := range m.issueDetail.Worklogs {
				total += wl.Time
			}
			if m.worklogTotals == nil {
				m.worklogTotals = make(map[string]int)
			}
			m.worklogTotals[m.issueDetail.ID] = total

			worklogsContent := m.buildWorklogsContent(m.detailLayout.rightColumnWidth - ui.PanelOverheadWidth)
			m.worklogsViewport.SetWidth(m.detailLayout.rightColumnWidth)
			m.worklogsViewport.SetHeight(m.detailLayout.worklogsHeight)
			m.worklogsViewport.SetContent(worklogsContent)
		}

		return m, nil

	case issueTypesLoadedMsg:
		m.loadingCount--
		m.issueTypes = msg.issueTypes
		return m, nil

	case transitionsLoadedMsg:
		m.loadingCount--
		m.transitions = msg.transitions
		m.transitionData = NewTransitionFormData(msg.transitions)
		return m, tea.Batch(m.transitionData.Form.Init())

	case statusesLoadedMsg:
		m.loadingCount--
		m.statuses = msg.statuses
		if len(m.statuses) > 0 {
			m.sections = m.classifyIssues(m.issues, m.statuses)
			m.listViewport.SetContent(m.buildListContent())
		}
		m.selectedIssue = m.sections[0].Issues[0]

		return m, nil

	case transitionPostedMsg:
		m.loadingCount--
		m.statusMessage.content = "Issue transitioned"
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		switch m.focusedSection {
		case metadataSection:
			m.loadingCount++
			cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		case subTasksSection:
			m.loadingCount++
			cmds = append(cmds, m.fetchSubTasksCmd(m.issueDetail.Key))
		}

		return m, tea.Batch(cmds...)

	case assigneePostedMsg:
		m.loadingCount--
		m.statusMessage.content = "User assigned successfully"
		return m, nil

	case newIssuePostedMsg:
		m.loadingCount--
		var cmds []tea.Cmd
		if m.issueDetail != nil {
			m.mode = detailView
			m.loadingCount++
			cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		} else {
			m.mode = listView
			m.loadingCount++
			cmds = append(cmds, m.fetchMyIssuesCmd())
		}
		m.statusMessage.content = "New issue created successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		return m, tea.Batch(cmds...)

	case issueLinkPostedMsg:
		m.loadingCount--
		m.statusMessage.content = "Issue liked successfully"
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case updatedDescriptionMsg:
		m.loadingCount--
		var cmds []tea.Cmd
		m.statusMessage.content = "Description edited successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case priorityPostedMsg:
		m.loadingCount--
		m.statusMessage.content = "Priority posted successfully"
		var cmds []tea.Cmd
		m.statusMessage.content = "Priority posted successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case commentPostedMsg:
		m.loadingCount--
		m.statusMessage.content = "Comment posted successfully"
		var cmds []tea.Cmd
		m.statusMessage.content = "Comment posted successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case commentUpdatedMsg:
		m.loadingCount--
		m.statusMessage.content = "Comment edited successfully"
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case commentDeletedMsg:
		m.loadingCount--
		m.statusMessage.content = "Comment deleted successfully"
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case workLogPostedMsg:
		m.loadingCount--
		m.statusMessage.content = "Worklog posted successfully"
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		m.loadingCount++
		cmds = append(cmds, m.fetchWorkLogsCmd(m.issueDetail.ID))
		return m, tea.Batch(cmds...)

	case workLogUpdatedMsg:
		m.loadingCount--
		m.statusMessage.content = "Worklog edited successfully"
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		m.loadingCount++
		cmds = append(cmds, m.fetchWorkLogsCmd(m.issueDetail.ID))
		return m, tea.Batch(cmds...)

	case workLogDeletedMsg:
		m.loadingCount--
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		m.loadingCount++
		cmds = append(cmds, m.fetchWorkLogsCmd(m.issueDetail.ID))
		return m, tea.Batch(cmds...)

	case estimatePostedMsg:
		m.loadingCount--
		m.statusMessage.content = "Estimate posted successfully"
		var cmds []tea.Cmd
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		if m.pendingTransition != nil {
			transition := m.pendingTransition
			if isCancelTransition(*transition) {
				m.cancelReasonData = NewCancelReasonFormData()
				m.mode = cancelReasonView
				return m, m.cancelReasonData.Form.Init()
			}
			m.pendingTransition = nil
			cmds = append(cmds, m.postTransitionCmd(m.issueDetail.Key, transition.ID, transition.Name))
			return m, tea.Batch(cmds...)
		}
		m.mode = detailView
		m.loadingCount++
		cmds = append(cmds, m.fetchIssueDetailCmd(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case assignableUsersLoadedMsg:
		m.usersCache = msg.users
		m.loadingCount--
		return m, nil

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
		m.statusMessage = statusMessage{
			"Fetching my issues...",
			infoStatusBarMsg,
		}
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		return m, tea.Batch(cmds...)

	case issueDetailPollMsg:
		var cmds []tea.Cmd
		if m.mode != detailView {
			return m, nil
		}
		m.loadingCount++
		detailCmd := m.fetchIssueDetailCmd(m.activeIssue.Key)
		cmds = append(cmds, detailCmd)
		cmds = append(cmds, tea.Tick(time.Minute, func(t time.Time) tea.Msg {
			return issueDetailPollMsg{}
		}))

		m.statusMessage = statusMessage{
			"Fetching issue details...",
			infoStatusBarMsg,
		}
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		return m, tea.Batch(cmds...)

	case spinner.TickMsg:
		var tickCmd tea.Cmd
		m.spinner, tickCmd = m.spinner.Update(msg)
		return m, tickCmd

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width

		if m.issueDetail != nil {
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
		log.Printf("ERROR: %s", msg.err)

		m.statusMessage.content = msg.err.Error()
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		if m.mode == issueSearchView && m.searchData != nil {
			m.searchData = NewSearchFormData()
			m.searchData.Err = msg.err
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
	case issueSearchView:
		tmpModel, viewCmd = m.updateSearchIssueView(msg)
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
	case issueSearchView:
		content = m.renderSearchIssueView()
	default:
		content = "Unknown view\n"
	}

	return tea.NewView(content)
}

func main() {

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
	log.SetOutput(logFile)

	textInput := textinput.New()
	textInput.CharLimit = 50

	textAreaBox := textarea.New()
	textAreaBox.CharLimit = 3000
	textAreaBox.MaxHeight = 20
	textAreaBox.MaxHeight = 80

	spinner := spinner.New()

	p := tea.NewProgram(model{
		mode:          listView,
		client:        client,
		textInput:     textInput,
		textArea:      textAreaBox,
		windowWidth:   80,
		windowHeight:  24,
		spinner:       spinner,
		worklogTotals: make(map[string]int),
		columnWidths:  ui.CalculateColumnWidths(80),
		loadingCount:  6, // Init cmds
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
