// cmd/jira-tui/main.go
package main

import (
	"fmt"
	"log"
	"os"

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
	err                    error
	sections               []Section
	activeSection          int
	sectionCursor          int
	statuses               []jira.Status
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

	switch msg := msg.(type) {

	case myselfLoadedMsg:
		m.myself = msg.me
		return m, nil

	case issuesLoadedMsg:
		m.issues = msg.issues
		m.loading = false
		if len(m.statuses) > 0 {
			m.classifyIssues()
		}
		return m, nil

	case prioritiesLoadedMsg:
		m.priorityOptions = msg.priorities
		m.loading = false
		return m, nil

	case issueDetailLoadedMsg:
		m.issueDetail = msg.detail
		m.loadingDetail = false

	case workLogsLoadedMSg:
		m.selectedIssueWorklogs = msg.workLogs
		m.loadingWorkLogs = false

	case transitionsLoadedMsg:
		m.transitions = msg.transitions
		m.loadingTransitions = false

	case statusesLoadedMsg:
		m.statuses = msg.statuses
		if len(m.statuses) > 0 {
			m.classifyIssues()
		}
		return m, nil

	case transitionCompleteMsg:
		m.mode = detailView
		m.loadingDetail = true
		m.issueDetail = nil
		return m, m.fetchIssueDetail(m.selectedIssue.Key)

	case editedDescriptionMsg:
		m.mode = detailView
		m.loadingDetail = true
		if m.selectedIssue != nil {
			return m, m.fetchIssueDetail(m.selectedIssue.Key)
		}
		return m, nil

	case editedPriorityMsg:
		m.mode = detailView
		m.loadingDetail = true
		if m.selectedIssue != nil {
			return m, m.fetchIssueDetail(m.selectedIssue.Key)
		}
		return m, nil

	case postedCommentMsg:
		m.mode = detailView
		m.loadingDetail = true
		if m.selectedIssue != nil {
			return m, m.fetchIssueDetail(m.selectedIssue.Key)
		}
		return m, nil

	case postedWorkLog:
		m.mode = detailView
		m.loadingDetail = true
		if m.selectedIssue != nil {
			return m, m.fetchIssueDetail(m.selectedIssue.Key)
		}

	case assignableUsersLoadedMsg:
		m.assignableUsersCache = msg.users
		m.loadingAssignableUsers = false
		m.mode = assignableUsersSearchView

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width

		if m.listViewport == nil {
			vp := viewport.New(m.windowWidth-4, m.windowHeight-3)
			m.listViewport = &vp
		} else {
			m.listViewport.Width = m.windowWidth - 4
			m.listViewport.Height = m.windowHeight - 3
		}

		if m.mode == detailView {
			headerHeight := 15 // NOTE: Adjust based on your header size
			footerHeight := 2  // NOTE: Adjust based on your footer size

			m.detailViewport.Width = msg.Width - 10
			m.detailViewport.Height = msg.Height - headerHeight - footerHeight
		}

		return m, nil

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.loadingDetail = false
		m.loadingTransitions = false

	}

	switch m.mode {
	case listView:
		return m.updateListView(msg)
	case detailView:
		return m.updateDetailView(msg)
	case editDescriptionView:
		return m.updateEditDescriptionView(msg)
	case editPriorityView:
		return m.updateEditPriorityView(msg)
	case transitionView:
		return m.updateTransitionView(msg)
	case postCommentView:
		return m.updatePostCommentView(msg)
	case assignableUsersSearchView:
		return m.updateAssignableUsersView(msg)
	case postWorklogView:
		return m.updatePostWorklogView(msg)
	}

	return m, nil
}

func (m model) View() string {
	var content string

	if m.loading {
		return "\033[H\033[2J" + "Loading issues...\n" // \033[2J clears screen
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

	p := tea.NewProgram(model{
		loading:      true,
		mode:         listView,
		client:       client,
		filterInput:  filterBox,
		editTextArea: editTextAreaBox,
		windowWidth:  80,
		windowHeight: 24,
		sections:     sections,
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}

// log.Printf("INIT CURSOR: cursor %d // sectionCursor %d // issues %d", m.cursor, m.sectionCursor, len(sectionIssues))
