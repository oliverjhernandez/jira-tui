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

type viewMode int

const clearMsgTimeout = 5 * time.Second

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
	estimateView
	cancelReasonView
	issueSearchView
)

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
	childrenSection
)

func (f focusedSection) String() string {
	switch f {
	case descriptionSection:
		return "descSection"
	case commentsSection:
		return "commentsSection"
	case worklogsSection:
		return "worklogsSection"
	case childrenSection:
		return "childrenSection"
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
	client *jira.Client
	mode   viewMode
	err    error

	// Window & Layout
	windowWidth      int
	windowHeight     int
	detailLayout     detailLayout
	columnWidths     ui.ColumnWidths
	listViewport     viewport.Model
	descViewport     viewport.Model
	commentsViewport viewport.Model
	worklogsViewport viewport.Model
	childrenViewport viewport.Model

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
	childrenCursor   int

	// Input Components
	textInput textinput.Model
	textArea  textarea.Model
	filtering bool
	lastKey   string

	// Editing State
	editingDescription bool
	editingPriority    bool
	editingComment     bool
	editingWorklog     bool

	// Form Data
	worklogFormData  *WorklogFormData
	newIssueData     *NewIssueFormData
	estimateData     *EstimateFormData
	searchData       *SearchIssueFormData
	commentData      *CommentFormData
	descriptionData  *DescriptionFormData
	priorityData     *PriorityFormData
	transitionData   *TransitionFormData
	cancelReasonData *CancelReasonFormData

	// Loading States
	loadingIssues      bool
	loadingDetail      bool
	loadingTransitions bool
	loadingAssignUsers bool
	loadingWorkLogs    bool

	// UI Elements
	spinner       spinner.Model
	statusMessage string
}

func (m model) Init() tea.Cmd {

	var cmds []tea.Cmd
	cmds = append(cmds, m.fetchMySelf())
	cmds = append(cmds, m.fetchProjects())
	cmds = append(cmds, m.fetchMyIssues())
	cmds = append(cmds, m.fetchPriorities())
	cmds = append(cmds, m.fetchAllUsers())
	cmds = append(cmds, m.fetchIssueTypes())

	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var spinnerCmd tea.Cmd
	if tickMsg, ok := msg.(spinner.TickMsg); ok {
		if m.loadingIssues || m.loadingDetail || m.loadingTransitions || m.loadingWorkLogs {
			m.spinner, spinnerCmd = m.spinner.Update(tickMsg)
		}
	}

	switch msg := msg.(type) {
	case myselfLoadedMsg:
		m.myself = msg.me
		return m, nil

	case issuesLoadedMsg:
		m.issues = msg.issues
		m.loadingIssues = false

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

			cmds = append(cmds, m.fetchStatuses(m.activeProjects))
		}
		cmds = append(cmds, m.fetchAllWorklogsTotal(msg.issues))

		return m, tea.Batch(cmds...)

	case childrenLoadedMsg:
		m.searchData = NewSearchFormData()
		if m.issueDetail != nil {
			m.issueDetail.Children = msg.children
		}

		childrenContent := m.buildChildrenContent(m.detailLayout.rightColumnWidth - 10)
		m.childrenViewport.SetWidth(m.detailLayout.rightColumnWidth)
		m.childrenViewport.SetHeight(m.detailLayout.childrenHeight)
		m.childrenViewport.SetContent(childrenContent)

		return m, nil

	case worklogTotalsLoadedMsg:
		if m.worklogTotals == nil {
			m.worklogTotals = make(map[string]int)
		}
		maps.Copy(m.worklogTotals, msg.totals)
		return m, nil

	case prioritiesLoadedMsg:
		m.priorities = msg.priorities
		m.loadingIssues = false
		return m, nil

	case projectsLoadedMsg:
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

			cmds = append(cmds, m.fetchStatuses(m.activeProjects))
		}

		return m, tea.Batch(cmds...)

	case issueDetailLoadedMsg:
		m.issueDetail = msg.detail
		m.detailLayout = m.calculateDetailLayout()
		m.loadingDetail = false
		m.mode = detailView

		if m.issueDetail != nil {
			m.commentsViewport.SetWidth(m.detailLayout.leftColumnWidth)
			descContent := m.buildDescriptionContent(m.detailLayout.leftColumnWidth)
			m.descViewport.SetHeight(m.detailLayout.descHeight)
			m.descViewport.SetWidth(m.detailLayout.leftColumnWidth - 6)
			m.descViewport.SetContent(descContent)

			commentsContent := m.buildCommentsContent(m.detailLayout.leftColumnWidth)
			m.commentsViewport.SetHeight(m.detailLayout.commentsHeight)
			m.commentsViewport.SetWidth(m.detailLayout.leftColumnWidth - 6)
			m.commentsViewport.SetContent(commentsContent)
		}

		var cmds []tea.Cmd
		worklogsCmd := m.fetchWorkLogs(m.issueDetail.ID)
		cmds = append(cmds, worklogsCmd)

		epicChildrenCmd := m.fetchEpicChildren(m.issueDetail.Key)
		cmds = append(cmds, epicChildrenCmd)

		return m, tea.Batch(cmds...)

	case workLogsLoadedMsg:
		m.loadingWorkLogs = false
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

			worklogsContent := m.buildWorklogsContent(m.detailLayout.rightColumnWidth - 10)
			m.worklogsViewport.SetWidth(m.detailLayout.rightColumnWidth)
			m.worklogsViewport.SetHeight(m.detailLayout.worklogsHeight)
			m.worklogsViewport.SetContent(worklogsContent)
		}

		return m, nil

	case issueTypesLoadedMsg:
		m.issueTypes = msg.issueTypes
		return m, nil

	case transitionsLoadedMsg:
		m.transitions = msg.transitions
		m.loadingTransitions = false
		m.transitionData = NewTransitionFormData(msg.transitions)
		return m, tea.Batch(m.transitionData.Form.Init())

	case statusesLoadedMsg:
		m.statuses = msg.statuses
		if len(m.statuses) > 0 {
			m.sections = m.classifyIssues(m.issues, m.statuses)
		}

		return m, nil

	case transitionCompleteMsg:
		var cmds []tea.Cmd
		m.statusMessage = "Issue transitioned successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		switch m.focusedSection {
		case metadataSection:
			m.loadingDetail = true
			cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		case childrenSection:
			cmds = append(cmds, m.fetchEpicChildren(m.issueDetail.Key))
		}

		return m, tea.Batch(cmds...)

	case newIssueCompleteMsg:
		var cmds []tea.Cmd
		if m.issueDetail != nil {
			m.mode = detailView
			cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		} else {
			m.mode = listView
			cmds = append(cmds, m.fetchMyIssues())
		}
		m.statusMessage = "New issue created successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		return m, tea.Batch(cmds...)

	case linkIssueCompleteMsg:
		var cmds []tea.Cmd
		m.statusMessage = "Issue liked successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingDetail = true
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case editedDescriptionMsg:
		var cmds []tea.Cmd
		m.statusMessage = "Description edited successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingDetail = true
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case editedPriorityMsg:
		var cmds []tea.Cmd
		m.statusMessage = "Priority posted successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingDetail = true
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case postedCommentMsg:
		var cmds []tea.Cmd
		m.statusMessage = "Comment posted successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingDetail = true
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case updatedCommentMsg:
		var cmds []tea.Cmd
		m.statusMessage = "Comment edited successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case deletedCommentMsg:
		var cmds []tea.Cmd
		m.statusMessage = "Comment deleted successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case postedWorkLog:
		var cmds []tea.Cmd
		m.statusMessage = "Worklog posted successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingDetail = true
		m.loadingWorkLogs = true
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		cmds = append(cmds, m.fetchWorkLogs(m.issueDetail.ID))
		return m, tea.Batch(cmds...)

	case editedWorkLog:
		var cmds []tea.Cmd
		m.statusMessage = "Worklog edited successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingDetail = true
		m.loadingWorkLogs = true
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		cmds = append(cmds, m.fetchWorkLogs(m.issueDetail.ID))
		return m, tea.Batch(cmds...)

	case deletedWorkLog:
		var cmds []tea.Cmd
		m.statusMessage = "Worklog deleted successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))
		m.mode = detailView
		m.loadingDetail = true
		m.loadingWorkLogs = true
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		cmds = append(cmds, m.fetchWorkLogs(m.issueDetail.ID))
		return m, tea.Batch(cmds...)

	case postedEstimateMsg:
		var cmds []tea.Cmd
		m.statusMessage = "Estimate posted successfully"
		cmds = append(cmds, m.clearStatusAfter(clearMsgTimeout))

		if m.pendingTransition != nil {
			transition := m.pendingTransition
			if isCancelTransition(*transition) {
				m.cancelReasonData = NewCancelReasonFormData()
				m.mode = cancelReasonView
				return m, m.cancelReasonData.Form.Init()
			}
			m.pendingTransition = nil
			cmds = append(cmds, m.postTransition(m.issueDetail.Key, transition.ID, transition.Name))
			return m, tea.Batch(cmds...)
		}
		m.mode = detailView
		m.loadingDetail = true
		cmds = append(cmds, m.fetchIssueDetail(m.issueDetail.Key))
		return m, tea.Batch(cmds...)

	case assignUsersLoadedMsg:
		m.loadingAssignUsers = false
		m.mode = userSearchView
		return m, nil

	case usersLoadedMsg:
		m.usersCache = msg.users
		return m, nil

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width

		if m.issueDetail != nil {
			m.detailLayout = m.calculateDetailLayout()

			descContent := m.buildDescriptionContent(m.detailLayout.leftColumnWidth - 10)
			m.descViewport.SetWidth(m.detailLayout.leftColumnWidth)
			m.descViewport.SetHeight(m.detailLayout.descHeight)
			m.descViewport.SetContent(descContent)

			commentsContent := m.buildCommentsContent(m.detailLayout.leftColumnWidth - 10)
			m.commentsViewport.SetWidth(m.detailLayout.leftColumnWidth)
			m.commentsViewport.SetHeight(m.detailLayout.commentsHeight)
			m.commentsViewport.SetContent(commentsContent)
		}

		m.columnWidths = ui.CalculateColumnWidths(msg.Width)

		// INFO: this should be calculated
		infoPanelHeight := 6
		m.listViewport.SetWidth(m.windowWidth - 4)
		m.listViewport.SetHeight(m.windowHeight - 3 - infoPanelHeight)

		return m, nil

	case keyTimeoutMsg:
		m.lastKey = ""
		return m, nil

	case clearStatusMsg:
		m.statusMessage = ""
		return m, nil

	case errMsg:
		var cmds []tea.Cmd

		m.loadingIssues = false
		m.loadingDetail = false
		m.loadingTransitions = false

		log.Printf("ERROR: %s", msg.err)

		m.statusMessage = msg.err.Error()
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
	case estimateView:
		tmpModel, viewCmd = m.updatePostEstimateView(msg)
	case cancelReasonView:
		tmpModel, viewCmd = m.updatePostCancelReasonView(msg)
	case issueSearchView:
		tmpModel, viewCmd = m.updateSearchIssueView(msg)
	}

	m = tmpModel.(model)
	return m, tea.Batch(spinnerCmd, viewCmd)
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
		loadingIssues:    true,
		mode:             listView,
		client:           client,
		textInput:        textInput,
		textArea:         textAreaBox,
		windowWidth:      80,
		windowHeight:     24,
		spinner:          spinner,
		worklogTotals:    make(map[string]int),
		columnWidths:     ui.CalculateColumnWidths(80),
		listViewport:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(40)),
		descViewport:     viewport.New(viewport.WithWidth(80), viewport.WithHeight(40)),
		commentsViewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(40)),
		worklogsViewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(40)),
		childrenViewport: viewport.New(viewport.WithWidth(80), viewport.WithHeight(40)),
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
