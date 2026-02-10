// cmd/jira-tui/main.go
package main

import (
	"fmt"
	"log"
	"maps"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/oliverjhernandez/jira-tui/internal/config"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

var Projects = []string{"DEV", "DCSDM", "ITELMEX", "EL"}

type viewMode int

const (
	listView viewMode = iota
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
	windowWidth    int
	windowHeight   int
	columnWidths   ui.ColumnWidths
	listViewport   *viewport.Model
	detailViewport *viewport.Model

	// User Data
	myself *jira.User

	// Issue Data
	issues        []jira.Issue
	selectedIssue *jira.Issue
	issueDetail   *jira.IssueDetail
	epicChildren  []jira.Issue

	// Issue Metadata
	sections         []Section
	filteredSections []Section
	statuses         []jira.Status
	priorityOptions  []jira.Priority

	// Worklogs
	selectedIssueWorklogs []jira.WorkLog
	worklogTotals         map[string]int

	// Transitions
	transitions       []jira.Transition
	pendingTransition *jira.Transition

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
	rightColumnView  rightColumnView

	// Input Components
	textInput textinput.Model
	textArea  textarea.Model
	filtering bool
	lastKey   string

	// Editing State
	editingDescription bool
	editingPriority    bool

	// Form Data
	worklogData      *WorklogFormData
	estimateData     *EstimateFormData
	searchData       *SearchIssueFormData
	commentData      *CommentFormData
	descriptionData  *DescriptionFormData
	priorityData     *PriorityFormData
	transitionData   *TransitionFormData
	cancelReasonData *CancelReasonFormData

	// Loading States
	loading            bool
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
	cmds = append(cmds, m.fetchMyIssues())
	cmds = append(cmds, m.fetchStatuses())
	cmds = append(cmds, m.fetchPriorities())

	return tea.Batch(cmds...)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var spinnerCmd tea.Cmd
	if tickMsg, ok := msg.(spinner.TickMsg); ok {
		if m.loading || m.loadingDetail || m.loadingTransitions || m.loadingWorkLogs {
			m.spinner, spinnerCmd = m.spinner.Update(tickMsg)
		}
	}

	switch msg := msg.(type) {
	case myselfLoadedMsg:
		m.myself = msg.me
		return m, nil

	case issuesLoadedMsg:
		m.issues = msg.issues
		m.loading = false
		if len(m.statuses) > 0 {
			m.sections = m.classifyIssues(m.issues, m.statuses)
		}
		return m, tea.Batch(m.fetchAllWorklogTotals(msg.issues))

	case epicChildrenLoadedMsg:
		m.epicChildren = nil
		m.epicChildren = msg.children
		return m, nil

	case worklogTotalsLoadedMsg:
		if m.worklogTotals == nil {
			m.worklogTotals = make(map[string]int)
		}
		maps.Copy(m.worklogTotals, msg.totals)
		return m, nil

	case prioritiesLoadedMsg:
		m.priorityOptions = msg.priorities
		m.loading = false
		return m, nil

	case issueDetailLoadedMsg:
		if m.detailViewport == nil {
			headerHeight := 15
			footerHeight := 1
			// NOTE: determine window
			width := m.windowWidth - 10
			height := m.windowHeight - headerHeight - footerHeight
			vp := viewport.New(width, height)
			m.detailViewport = &vp
		}

		if m.searchData != nil {
			m.searchData = NewSearchFormData()
		}

		m.issueDetail = msg.detail
		m.loadingDetail = false
		m.mode = detailView
		return m, nil

	case workLogsLoadedMsg:
		m.selectedIssueWorklogs = msg.workLogs
		m.loadingWorkLogs = false
		if m.issueDetail != nil {
			var total int
			for _, wl := range msg.workLogs {
				total += wl.Time
			}
			if m.worklogTotals == nil {
				m.worklogTotals = make(map[string]int)
			}
			m.worklogTotals[m.issueDetail.ID] = total
		}
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
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(m.fetchIssueDetail(m.issueDetail.Key))

	case linkIssueCompleteMsg:
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(m.fetchIssueDetail(m.issueDetail.Key))

	case editedDescriptionMsg:
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(m.fetchIssueDetail(m.issueDetail.Key))

	case editedPriorityMsg:
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(m.fetchIssueDetail(m.issueDetail.Key))

	case postedCommentMsg:
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(m.fetchIssueDetail(m.issueDetail.Key))

	case postedWorkLog:
		m.mode = detailView
		m.loadingDetail = true
		m.loadingWorkLogs = true
		return m, tea.Batch(m.fetchIssueDetail(m.issueDetail.Key), m.fetchWorkLogs(m.issueDetail.ID))

	case postedEstimateMsg:
		if m.pendingTransition != nil {
			transition := m.pendingTransition
			if isCancelTransition(*transition) {
				m.cancelReasonData = NewCancelReasonFormData()
				m.mode = cancelReasonView
				return m, m.cancelReasonData.Form.Init()
			}
			m.pendingTransition = nil
			return m, tea.Batch(m.postTransition(m.issueDetail.Key, transition.ID))
		}
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(m.fetchIssueDetail(m.issueDetail.Key))

	case assignUsersLoadedMsg:
		m.usersCache = msg.users
		m.loadingAssignUsers = false
		m.mode = userSearchView
		return m, nil

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width
		m.columnWidths = ui.CalculateColumnWidths(msg.Width)

		infoPanelHeight := 6
		if m.listViewport == nil {
			vp := viewport.New(m.windowWidth-4, m.windowHeight-3-infoPanelHeight)
			m.listViewport = &vp
		} else {
			m.listViewport.Width = m.windowWidth - 4
			m.listViewport.Height = m.windowHeight - 3 - infoPanelHeight
		}

		if m.detailViewport != nil {
			headerHeight := 15
			footerHeight := 1
			m.detailViewport.Width = msg.Width - 10
			m.detailViewport.Height = msg.Height - headerHeight - footerHeight
		}
		return m, nil

	case keyTimeoutMsg:
		m.lastKey = ""
		return m, nil

	case errMsg:
		log.Printf("ERROR: %s", msg.err)
		m.loading = false
		m.loadingDetail = false
		m.loadingTransitions = false

		if m.mode == issueSearchView && m.searchData != nil {
			m.searchData = NewSearchFormData()
			m.searchData.Err = msg.err
		}

		log.Printf("ERROR: %w", msg.err)

		return m, nil
	}

	var viewCmd tea.Cmd
	var tmpModel tea.Model
	switch m.mode {
	case listView:
		tmpModel, viewCmd = m.updateListView(msg)
	case detailView:
		tmpModel, viewCmd = m.updateDetailView(msg)
	case descriptionView:
		tmpModel, viewCmd = m.updateEditDescriptionView(msg)
	case priorityView:
		tmpModel, viewCmd = m.updateEditPriorityView(msg)
	case transitionView:
		tmpModel, viewCmd = m.updateTransitionView(msg)
	case commentView:
		tmpModel, viewCmd = m.updatePostCommentView(msg)
	case userSearchView:
		tmpModel, viewCmd = m.updateSearchUserView(msg)
	case worklogView:
		tmpModel, viewCmd = m.updatePostWorklogView(msg)
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

func (m model) View() string {
	var content string

	if m.loading {
		return "\033[H\033[2J"

	}
	if m.err != nil {
		return "\033[H\033[2J" + fmt.Sprintf("Error: %v\n\nPress 'q' to quit.\n", m.err)
	}

	switch m.mode {
	case listView:
		content = m.renderListView()
	case detailView:
		content = m.renderDetailView()
	case transitionView:
		content = m.renderTransitionView()
	case descriptionView:
		content = m.renderEditDescriptionView()
	case priorityView:
		content = m.renderEditPriorityView()
	case commentView:
		content = m.renderPostCommentView()
	case userSearchView:
		content = m.renderSearchUserView()
	case worklogView:
		content = m.renderPostWorklogView()
	case estimateView:
		content = m.renderPostEstimateView()
	case cancelReasonView:
		content = m.renderPostCancelReasonView()
	case issueSearchView:
		content = m.renderSearchIssueView()
	default:
		content = "Unknown view\n"
	}

	return "\033[H" + content
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
		loading:       true,
		mode:          listView,
		client:        client,
		textInput:     textInput,
		textArea:      textAreaBox,
		windowWidth:   80,
		windowHeight:  24,
		spinner:       spinner,
		worklogTotals: make(map[string]int),
		columnWidths:  ui.CalculateColumnWidths(80),
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// log.Printf("INIT CURSOR: cursor %d // sectionCursor %d // issues %d", m.cursor, m.sectionCursor, len(sectionIssues))
