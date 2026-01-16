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

type viewMode int

type model struct {
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
	assignableUsersCache   []jira.User
	filteredUsers          []*jira.User
	assigneeCursor         int
	worklogData            *WorklogFormData
	err                    error
}

func (m model) Init() tea.Cmd {
	return tea.Batch(m.fetchMyIssues(), m.fetchPriorities)
	cmds = append(cmds, m.fetchStatuses())
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case issuesLoadedMsg:
		log.Printf("issuesLoadedMsg received - %d issues, current mode: %v", len(msg.issues), m.mode)
		m.issues = msg.issues
		m.loading = false
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

	case assignableUsersLoadedMsg:
		m.assignableUsersCache = msg.users
		m.loadingAssignableUsers = false
		m.mode = assignableUsersSearchView

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width

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
	}

	return m, nil
}

func (m model) View() string {
	var content string

	if m.loading {
		return "Loading issues...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.\n", m.err)
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
	default:
		content = "Unknown view\n"
	}

	return content
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

	p := tea.NewProgram(model{
		loading:      true,
		mode:         listView,
		client:       client,
		filterInput:  filterBox,
		editTextArea: editTextAreaBox,
		windowWidth:  80,
		windowHeight: 24,
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
