// cmd/jira-tui/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

type viewMode int

const (
	listView viewMode = iota
	detailView
	transitionView
)

type model struct {
	issues             []jira.Issue
	cursor             int
	loading            bool
	err                error
	mode               viewMode
	selectedIssue      *jira.Issue
	issueDetail        *jira.IssueDetail
	loadingDetail      bool
	client             *jira.Client
	transitions        []jira.Transition
	transitionCursor   int
	loadingTransitions bool
	filterInput        textinput.Model
	filtering          bool
}

// bubbletea messages from commands
type issuesLoadedMsg struct {
	issues []jira.Issue
}

type issueDetailLoadedMsg struct {
	detail *jira.IssueDetail
}

type transitionsLoadedMsg struct {
	transitions []jira.Transition
}

type transitionCompleteMsg struct {
	success bool
}

type errMsg struct {
	err error
}

func (m model) Init() tea.Cmd {
	return fetchIssues
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
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

	case issuesLoadedMsg:
		m.issues = msg.issues
		m.loading = false

	case issueDetailLoadedMsg:
		m.issueDetail = msg.detail
		m.loadingDetail = false

	case transitionsLoadedMsg:
		m.transitions = msg.transitions
		m.loadingTransitions = false

	case transitionCompleteMsg:
		// Transition completed, refresh the issue detail
		m.mode = detailView
		m.loadingDetail = true
		m.issueDetail = nil
		return m, m.fetchIssueDetail(m.selectedIssue.Key)

	case errMsg:
		m.err = msg.err
		m.loading = false
		m.loadingDetail = false
		m.loadingTransitions = false
	}

	return m, nil
}

func (m model) updateListView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {

	issuesToShow := m.issues
	if m.filterInput.Value() != "" {
		issuesToShow = filterIssues(m.issues, m.filterInput.Value())
	}

	if m.filtering {
		switch msg.String() {
		case "esc":
			m.filtering = false
			m.filterInput.SetValue("")
			m.filterInput.Blur()
			m.cursor = 0
			return m, nil
		case "enter":
			m.filtering = false
			m.filterInput.Blur()
			return m, nil
		}

		var cmd tea.Cmd
		m.filterInput, cmd = m.filterInput.Update(msg)

		return m, cmd
	}

	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}
	case "down", "j":
		if m.cursor < len(issuesToShow)-1 {
			m.cursor++
		}
	case "esc":
		m.filterInput.SetValue("")
		m.cursor = 0
	case "/":
		m.filtering = true
		m.filterInput.SetValue("")
		m.filterInput.Focus()
		m.cursor = 0
		return m, textinput.Blink
	case "enter":
		if len(issuesToShow) > 0 && m.cursor < len(issuesToShow) {
			m.selectedIssue = &issuesToShow[m.cursor]
			m.mode = detailView
			m.loadingDetail = true
			m.issueDetail = nil
			return m, m.fetchIssueDetail(m.selectedIssue.Key)
		}
	}

	return m, nil
}

func (m model) updateDetailView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc":
		m.mode = listView
		m.selectedIssue = nil
		m.issueDetail = nil
	case "t":
		if m.selectedIssue != nil {
			m.mode = transitionView
			m.loadingTransitions = true
			m.transitionCursor = 0
			return m, m.fetchTransitions(m.selectedIssue.Key)
		}
	}

	return m, nil
}

func (m model) updateTransitionView(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "backspace":
		// Go back to detail view
		m.mode = detailView
		m.transitions = nil
	case "up", "k":
		if m.transitionCursor > 0 {
			m.transitionCursor--
		}
	case "down", "j":
		if m.transitionCursor < len(m.transitions)-1 {
			m.transitionCursor++
		}
	case "enter":
		// Execute the selected transition
		if len(m.transitions) > 0 {
			transition := m.transitions[m.transitionCursor]
			return m, m.doTransition(m.selectedIssue.Key, transition.ID)
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.loading {
		return "Loading issues...\n"
	}

	if m.err != nil {
		return fmt.Sprintf("Error: %v\n\nPress 'q' to quit.\n", m.err)
	}

	switch m.mode {
	case listView:
		return m.renderListView()
	case detailView:
		return m.renderDetailView()
	case transitionView:
		return m.renderTransitionView()
	default:
		return "Unknown view\n"
	}
}

func (m model) renderListView() string {
	var b strings.Builder
	b.WriteString("My Jira Issues\n\n")

	issuesToShow := m.issues
	if m.filterInput.Value() != "" {
		issuesToShow = filterIssues(m.issues, m.filterInput.Value())
	}

	for i, issue := range issuesToShow {
		cursor := " "
		if m.cursor == i {
			cursor = ">"
		}

		b.WriteString(fmt.Sprintf("%s [%s] %s - %s\n",
			cursor, issue.Key, issue.Summary, issue.Status))
	}

	b.WriteString("\n")

	if m.filtering {
		b.WriteString("Filter: ")
		b.WriteString(m.filterInput.View())
		b.WriteString(" (enter to finish, esc to cancel)")
	} else if m.filterInput.Value() != "" {
		b.WriteString(fmt.Sprintf("Filtered by: '%s' (%d/%d) | / to change | esc to clear", m.filterInput.Value(), len(issuesToShow), len(m.issues)))
	} else {
		b.WriteString("\n/ filter | enter detail | t transition | q quit")
	}

	return b.String()
}

func (m model) renderDetailView() string {
	if m.selectedIssue == nil {
		return "No issue selected\n"
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("Issue: %s\n", m.selectedIssue.Key))
	b.WriteString(strings.Repeat("=", 50) + "\n\n")

	if m.loadingDetail {
		b.WriteString("Loading details...\n")
	} else if m.issueDetail != nil {
		b.WriteString(fmt.Sprintf("Summary: %s\n", m.issueDetail.Summary))
		b.WriteString(fmt.Sprintf("Type: %s\n", m.issueDetail.Type))
		b.WriteString(fmt.Sprintf("Status: %s\n", m.issueDetail.Status))
		b.WriteString(fmt.Sprintf("Assignee: %s\n", m.issueDetail.Assignee))
		b.WriteString(fmt.Sprintf("Reporter: %s\n", m.issueDetail.Reporter))
		b.WriteString("\n")

		if m.issueDetail.Description != "" {
			b.WriteString("Description:\n")
			b.WriteString(m.issueDetail.Description)
			b.WriteString("\n\n")
		}

		if len(m.issueDetail.Comments) > 0 {
			b.WriteString(fmt.Sprintf("Comments (%d):\n", len(m.issueDetail.Comments)))
			b.WriteString(strings.Repeat("-", 50) + "\n")
			for _, comment := range m.issueDetail.Comments {
				b.WriteString(fmt.Sprintf("%s - %s:\n%s\n\n",
					comment.Author, comment.Created, comment.Body))
			}
		}
	}

	b.WriteString("Press 't' to change status, Esc to go back, q to quit.\n")

	return b.String()
}

func (m model) renderTransitionView() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Change Status for %s\n", m.selectedIssue.Key))
	b.WriteString(strings.Repeat("=", 50) + "\n\n")

	if m.loadingTransitions {
		b.WriteString("Loading available transitions...\n")
	} else if len(m.transitions) == 0 {
		b.WriteString("No transitions available for this issue.\n")
	} else {
		b.WriteString("Select new status:\n\n")
		for i, transition := range m.transitions {
			cursor := " "
			if m.transitionCursor == i {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s %s\n", cursor, transition.Name))
		}
	}

	b.WriteString("\nPress j/k or ↑/↓ to navigate, Enter to select, Esc to cancel.\n")

	return b.String()
}

func (m model) fetchIssueDetail(issueKey string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		detail, err := m.client.GetIssueDetail(context.Background(), issueKey)
		if err != nil {
			return errMsg{err}
		}

		return issueDetailLoadedMsg{detail}
	}
}

func (m model) fetchTransitions(issueKey string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		transitions, err := m.client.GetTransitions(context.Background(), issueKey)
		if err != nil {
			return errMsg{err}
		}

		return transitionsLoadedMsg{transitions}
	}
}

func (m model) doTransition(issueKey, transitionID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.DoTransition(context.Background(), issueKey, transitionID)
		if err != nil {
			return errMsg{err}
		}

		return transitionCompleteMsg{success: true}
	}
}

func fetchIssues() tea.Msg {
	url := os.Getenv("JIRA_URL")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_TOKEN")

	if url == "" || email == "" || token == "" {
		return errMsg{fmt.Errorf("missing env vars: JIRA_URL, JIRA_EMAIL, JIRA_TOKEN")}
	}

	client, err := jira.NewClient(url, email, token)
	if err != nil {
		return errMsg{err}
	}

	issues, err := client.GetMyIssues(context.Background())
	if err != nil {
		return errMsg{err}
	}

	return issuesLoadedMsg{issues}
}

func filterIssues(issues []jira.Issue, filter string) []jira.Issue {
	var filtered []jira.Issue

	for _, i := range issues {
		if issueMatchesFilter(i, filter) {
			filtered = append(filtered, i)
		}
	}

	return filtered
}

func issueMatchesFilter(issue jira.Issue, filter string) bool {
	filterLower := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(issue.Summary), filterLower) ||
		strings.Contains(strings.ToLower(issue.Key), filterLower) ||
		strings.Contains(strings.ToLower(issue.Status), filterLower)
}

func main() {
	url := os.Getenv("JIRA_URL")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_TOKEN")

	client, _ := jira.NewClient(url, email, token)

	filterBox := textinput.New()
	filterBox.CharLimit = 50

	p := tea.NewProgram(model{
		loading:     true,
		mode:        listView,
		client:      client,
		filterInput: filterBox,
	})

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
