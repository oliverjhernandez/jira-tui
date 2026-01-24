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
)

var Projects = []string{"DEV", "DCSDM", "ITELMEX", "EL"}

type viewMode int

type Section struct {
	Name        string
	CategoryKey string
	Collapsed   bool
	Issues      []*jira.Issue
}

type model struct {
	myself                 *jira.User
	issues                 []jira.Issue
	mode                   viewMode
	cursor                 int
	priorityCursor         int
	transitionCursor       int
	priorityOptions        []jira.Priority
	loading                bool
	loadingDetail          bool
	loadingTransitions     bool
	loadingAssignableUsers bool
	loadingWorkLogs        bool
	selectedIssue          *jira.Issue
	selectedIssueWorklogs  []jira.WorkLog
	issueDetail            *jira.IssueDetail
	client                 *jira.Client
	transitions            []jira.Transition
	filterInput            textinput.Model
	filtering              bool
	editTextArea           textarea.Model
	editingDescription     bool
	editingPriority        bool
	postingComment         bool
	postingWorkLog         bool
	windowWidth            int
	windowHeight           int
	detailViewport         *viewport.Model
	listViewport           *viewport.Model
	assignableUsersCache   []jira.User
	filteredUsers          []*jira.User
	assigneeCursor         int
	worklogData            *WorklogFormData
	estimateData           *EstimateFormData
	pendingTransition      *jira.Transition
	err                    error
	sections               []Section
	sectionCursor          int
	statuses               []jira.Status
	spinner                spinner.Model
	worklogTotals          map[string]int
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
	m.spinner, spinnerCmd = m.spinner.Update(msg)

	switch msg := msg.(type) {

	case myselfLoadedMsg:
		m.myself = msg.me
		return m, spinnerCmd

	case issuesLoadedMsg:
		m.issues = msg.issues
		m.loading = false
		if len(m.statuses) > 0 {
			m.classifyIssues()
		}
		return m, tea.Batch(spinnerCmd, m.fetchAllWorklogTotals(msg.issues))

	case worklogTotalsLoadedMsg:
		if m.worklogTotals == nil {
			m.worklogTotals = make(map[string]int)
		}
		maps.Copy(m.worklogTotals, msg.totals)
		return m, spinnerCmd

	case prioritiesLoadedMsg:
		m.priorityOptions = msg.priorities
		m.loading = false
		return m, spinnerCmd

	case issueDetailLoadedMsg:
		m.issueDetail = msg.detail
		m.loadingDetail = false
		m.mode = detailView
		return m, spinnerCmd

	case workLogsLoadedMSg:
		m.selectedIssueWorklogs = msg.workLogs
		m.loadingWorkLogs = false
		// Update worklogTotals map to keep it in sync
		if m.selectedIssue != nil {
			var total int
			for _, wl := range msg.workLogs {
				total += wl.Time
			}
			if m.worklogTotals == nil {
				m.worklogTotals = make(map[string]int)
			}
			m.worklogTotals[m.selectedIssue.ID] = total
		}
		return m, spinnerCmd

	case transitionsLoadedMsg:
		m.transitions = msg.transitions
		m.loadingTransitions = false
		return m, spinnerCmd

	case statusesLoadedMsg:
		m.statuses = msg.statuses
		if len(m.statuses) > 0 {
			m.classifyIssues()
		}
		return m, spinnerCmd

	case transitionCompleteMsg:
		m.mode = detailView
		m.loadingDetail = true
		m.issueDetail = nil
		return m, tea.Batch(spinnerCmd, m.fetchIssueDetail(m.selectedIssue.Key))

	case editedDescriptionMsg:
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(spinnerCmd, m.fetchIssueDetail(m.selectedIssue.Key))

	case editedPriorityMsg:
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(spinnerCmd, m.fetchIssueDetail(m.selectedIssue.Key))

	case postedCommentMsg:
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(spinnerCmd, m.fetchIssueDetail(m.selectedIssue.Key))

	case postedWorkLog:
		m.mode = detailView
		m.loadingDetail = true
		m.loadingWorkLogs = true
		return m, tea.Batch(spinnerCmd, m.fetchIssueDetail(m.issueDetail.Key), m.fetchWorkLogs(m.selectedIssue.ID))

	case postedEstimateMsg:
		if m.pendingTransition != nil {
			transition := m.pendingTransition
			if isCancelTransition(*transition) {
				m.editTextArea.Reset()
				m.editTextArea.Focus()
				m.mode = postCancelReasonView
				return m, textarea.Blink
			}
			m.pendingTransition = nil
			return m, tea.Batch(spinnerCmd, m.postTransition(m.selectedIssue.Key, transition.ID))
		}
		m.mode = detailView
		m.loadingDetail = true
		return m, tea.Batch(spinnerCmd, m.fetchIssueDetail(m.selectedIssue.Key))

	case assignableUsersLoadedMsg:
		m.assignableUsersCache = msg.users
		m.loadingAssignableUsers = false
		m.mode = assignableUsersSearchView
		return m, spinnerCmd

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width

		infoPanelHeight := 5 // 2 content lines + 2 border lines + 1 newline
		if m.listViewport == nil {
			vp := viewport.New(m.windowWidth-4, m.windowHeight-3-infoPanelHeight)
			m.listViewport = &vp
		} else {
			m.listViewport.Width = m.windowWidth - 4
			m.listViewport.Height = m.windowHeight - 3 - infoPanelHeight
		}

		if m.mode == detailView {
			headerHeight := 15 // NOTE: Adjust based on your header size
			footerHeight := 1  // NOTE: Adjust based on your footer size

			m.detailViewport.Width = msg.Width - 10
			m.detailViewport.Height = msg.Height - headerHeight - footerHeight
		}

		return m, spinnerCmd

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.loadingDetail = false
		m.loadingTransitions = false

		return m, spinnerCmd
	}

	var viewCmd tea.Cmd
	switch m.mode {
	case listView:
		var tmp tea.Model
		tmp, viewCmd = m.updateListView(msg)
		m = tmp.(model)
	case detailView:
		var tmp tea.Model
		tmp, viewCmd = m.updateDetailView(msg)
		m = tmp.(model)
	case editDescriptionView:
		var tmp tea.Model
		tmp, viewCmd = m.updateEditDescriptionView(msg)
		m = tmp.(model)
	case editPriorityView:
		var tmp tea.Model
		tmp, viewCmd = m.updateEditPriorityView(msg)
		m = tmp.(model)
	case transitionView:
		var tmp tea.Model
		tmp, viewCmd = m.updateTransitionView(msg)
		m = tmp.(model)
	case postCommentView:
		var tmp tea.Model
		tmp, viewCmd = m.updatePostCommentView(msg)
		m = tmp.(model)
	case assignableUsersSearchView:
		var tmp tea.Model
		tmp, viewCmd = m.updateAssignableUsersView(msg)
		m = tmp.(model)
	case postWorklogView:
		var tmp tea.Model
		tmp, viewCmd = m.updatePostWorklogView(msg)
		m = tmp.(model)
	case postEstimateView:
		var tmp tea.Model
		tmp, viewCmd = m.updatePostEstimateView(msg)
		m = tmp.(model)
	case postCancelReasonView:
		var tmp tea.Model
		tmp, viewCmd = m.updatePostCancelReasonView(msg)
		m = tmp.(model)
	}

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
	case editDescriptionView:
		content = m.renderEditDescriptionView()
	case editPriorityView:
		content = m.renderEditPriorityView()
	case postCommentView:
		content = m.renderPostCommentView()
	case assignableUsersSearchView:
		content = m.renderAssignableUsersView()
	case postWorklogView:
		content = m.renderPostWorklogView()
	case postEstimateView:
		content = m.renderPostEstimateView()
	case postCancelReasonView:
		content = m.renderPostCancelReasonView()
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

	filterBox := textinput.New()
	filterBox.CharLimit = 50

	editTextAreaBox := textarea.New()
	editTextAreaBox.CharLimit = 3000
	editTextAreaBox.MaxHeight = 20
	editTextAreaBox.MaxHeight = 80

	sections := []Section{
		{Name: "In Progress", CategoryKey: "indeterminate"},
		{Name: "To Do", CategoryKey: "new"},
		{Name: "Done", CategoryKey: "done", Collapsed: true},
	}

	spinner := spinner.New()

	p := tea.NewProgram(model{
		loading:       true,
		mode:          listView,
		client:        client,
		filterInput:   filterBox,
		editTextArea:  editTextAreaBox,
		windowWidth:   80,
		windowHeight:  24,
		sections:      sections,
		spinner:       spinner,
		worklogTotals: make(map[string]int),
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// log.Printf("INIT CURSOR: cursor %d // sectionCursor %d // issues %d", m.cursor, m.sectionCursor, len(sectionIssues))
