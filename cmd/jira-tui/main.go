// cmd/jira-tui/main.go
package main

import (
	"fmt"
	"log"
	"os"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
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
	windowWidth        int
	windowHeight       int
}

func (m model) Init() tea.Cmd {
	return m.fetchData
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.mode {
	case editDescriptionView:
		return m.updateEditDescriptionView(msg)
	case editPriorityView:
		return m.updateEditPriorityView(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch m.mode {
		case listView:
			return m.updateListView(msg)
		case detailView:
			return m.updateDetailView(msg)
		case transitionView:
			return m.updateTransitionView(msg)
		}

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

	case tea.WindowSizeMsg:
		m.windowHeight = msg.Height
		m.windowWidth = msg.Width

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.loadingDetail = false
		m.loadingTransitions = false
	}

	return m, nil
}

func (m model) View() tea.View {
	var content string

	if m.loading {
		layer := lipgloss.NewLayer("Loading issues...\n")
		return tea.NewView(layer)
	}

	if m.err != nil {
		layer := lipgloss.NewLayer(fmt.Sprintf("Error: %v\n\nPress 'q' to quit.\n", m.err))
		return tea.NewView(layer)
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
	default:
		content = "Unknown view\n"
	}

	layer := lipgloss.NewLayer(content)
	return tea.NewView(layer)
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
