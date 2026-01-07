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
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

type viewMode int

type model struct {
	issues             []jira.Issue
	cursor             int
	priorityCursor     int
	transitionCursor   int
	priorityOptions    []jira.Priority
	loading            bool
	err                error
	mode               viewMode
	selectedIssue      *jira.Issue
	issueDetail        *jira.IssueDetail
	loadingDetail      bool
	client             *jira.Client
	transitions        []jira.Transition
	loadingTransitions bool
	filterInput        textinput.Model
	filtering          bool
	editTextArea       textarea.Model
	editingDescription bool
	editingPriority    bool
	postingComment     bool
	windowWidth        int
	windowHeight       int
	detailViewport     *viewport.Model
}

func (m model) Init() tea.Cmd {
	return m.fetchData
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	switch msg := msg.(type) {
	case dataLoadedMsg:
		m.issues = msg.issues
		m.priorityOptions = msg.priorities
		m.loading = false

	case issueDetailLoadedMsg:
		m.issueDetail = msg.detail
		m.loadingDetail = false

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
	case detailView:
		return m.updateDetailView(msg)
	case editDescriptionView:
		return m.updateEditDescriptionView(msg)
	case editPriorityView:
		return m.updateEditPriorityView(msg)
	case postCommentView:
		return m.updatePostCommentView(msg)
	}

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch m.mode {
		case listView:
			return m.updateListView(keyMsg) // NOTE: why expect keymsg?
		case transitionView:
			return m.updateTransitionView(keyMsg) // NOTE: why expect keymsg?
		}
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
	default:
		content = "Unknown view\n"
	}

	return content
}

func main() {
	url := os.Getenv("JIRA_URL")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_TOKEN")

	client, _ := jira.NewClient(url, email, token)

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
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
