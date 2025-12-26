// cmd/jira-tui/main.go
package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

var (
	primaryColor   = lipgloss.Color("15")  // Bright white (adapts)
	secondaryColor = lipgloss.Color("240") // Gray
	accentColor    = lipgloss.Color("42")  // Green

	listPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(primaryColor).
			Padding(1, 2).
			Height(20).
			Width(200)

	detailPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(secondaryColor).
				Padding(1, 2).
				Height(20).
				Width(100)

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(accentColor).
				Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("63"))

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Bold(true)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	// Field Styles
	keyFieldStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true).
			Width(12).
			Align(lipgloss.Left)

	statusFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(12).
				Align(lipgloss.Left)

	summaryFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(40).
				Align(lipgloss.Left)

	assigneeFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(20).
				Align(lipgloss.Left)

	priorityFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(10).
				Align(lipgloss.Left)

	statusInProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("42")).
				Padding(0, 1).
				Bold(true)

	statusDoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("42")).
			Padding(0, 1).
			Bold(true)

	statusToDoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("153")).
			Padding(0, 1).
			Bold(true)

	statusDefaultStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("240")).
				Padding(0, 1).
				Bold(true)
)

type viewMode int

const (
	listView viewMode = iota
	detailView
	transitionView
	editDescriptionView
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
	editTextArea       textarea.Model
	editingDescription bool
	windowWidth        int
	windowHeight       int
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

type editedDescriptionMsg struct {
	success bool
}

type errMsg struct {
	err error
}

func (m model) Init() tea.Cmd {
	return fetchIssues
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.mode == editDescriptionView {
		return m.updateEditDescriptionView(msg)
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
		m.mode = detailView
		m.loadingDetail = true
		m.issueDetail = nil
		return m, m.fetchIssueDetail(m.selectedIssue.Key)

	case editedDescriptionMsg:
		m.mode = listView
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
			if m.cursor < len(issuesToShow) {
				return m, m.fetchIssueDetail(issuesToShow[m.cursor].Key)
			}
		}
	case "down", "j":
		if m.cursor < len(issuesToShow)-1 {
			m.cursor++
			if m.cursor < len(issuesToShow) {
				return m, m.fetchIssueDetail(issuesToShow[m.cursor].Key)
			}
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
	case "e":
		m.mode = editDescriptionView
		m.editingDescription = true
		m.editTextArea.SetValue(m.issueDetail.Description)
		m.editTextArea.Focus()
		return m, textarea.Blink
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
	default:
		content = "Unknown view\n"
	}

	layer := lipgloss.NewLayer(content)
	return tea.NewView(layer)
}

func (m model) renderListView() string {
	var b strings.Builder
	b.WriteString("My Jira Issues\n\n")

	issuesToShow := m.issues
	if m.filterInput.Value() != "" {
		issuesToShow = filterIssues(m.issues, m.filterInput.Value())
	}

	var listContent strings.Builder
	for i, issue := range issuesToShow {
		key := keyFieldStyle.Render(fmt.Sprintf("[%s]", issue.Key))
		summary := summaryFieldStyle.Render(truncate(issue.Summary, 40))
		statusBadge := statusFieldStyle.Render(renderStatusBadge(issue.Status))
		// assignee := assigneeFieldStyle.Render(issue.Assignee)
		// priority := assigneeFieldStyle.Render(issue.Priority)

		line := key + " " + summary + " " + statusBadge

		if m.cursor == i {
			line = "> " + line
		} else {
			line = " " + line
		}

		listContent.WriteString(line + "\n")
	}

	var statusBar string
	if m.filtering {
		statusBar = "Filter: " + m.filterInput.View() + " (enter to finish, esc to cancel)"
	} else if m.filterInput.Value() != "" {
		statusBar = fmt.Sprintf("Filtered by: '%s' (%d/%d) | / to change | esc to clear", m.filterInput.Value(), len(issuesToShow), len(m.issues))
	} else {
		statusBar = "\n/ filter | enter detail | t transition | q quit"
	}

	listRender := listPanelStyle.Render(listContent.String())
	statusBarRender := statusBarStyle.Render(statusBar)

	return listRender + "\n" + statusBarRender
}

func (m model) renderDetailView() string {
	if m.selectedIssue == nil {
		return "No issue selected\n"
	}

	var detailContent strings.Builder
	selectedIssue := m.issues[m.cursor]

	header := detailHeaderStyle.Render(selectedIssue.Key) + " " + renderStatusBadge(selectedIssue.Status)
	detailContent.WriteString(header + "\n\n")
	detailContent.WriteString(renderField("Summary", truncate(selectedIssue.Summary, 40)) + "\n")
	detailContent.WriteString(renderField("Type", selectedIssue.Type) + "\n")

	if m.loadingDetail {
		detailContent.WriteString("Loading details...\n")
	} else if m.issueDetail != nil {

		if m.issueDetail != nil && m.issueDetail.Key == selectedIssue.Key {
			detailContent.WriteString(renderField("Assignee", m.issueDetail.Assignee) + "\n")
			detailContent.WriteString(renderField("Reporter", m.issueDetail.Reporter) + "\n")

			if m.issueDetail.Description != "" {
				detailContent.WriteString(detailLabelStyle.Render("Description:") + "\n")
				desc := m.issueDetail.Description
				if len(desc) > 200 {
					desc = desc[:200] + "..."
				}
				detailContent.WriteString(detailValueStyle.Render(desc) + "\n\n")
			}

			if len(m.issueDetail.Comments) > 0 {
				detailContent.WriteString(detailLabelStyle.Render(fmt.Sprintf("Comments: (%d):", len(m.issueDetail.Comments))) + "\n")
				detailContent.WriteString(detailValueStyle.Render("Press Enter for full view") + "\n")
			}
		} else {
			detailContent.WriteString("\n" + lipgloss.NewStyle().Faint(true).Render("Press Enter for full details") + "\n")
		}
	}

	var statusBar string
	if m.filtering {
		statusBar = "Filter: " + m.filterInput.View() + " (enter to finish, esc to cancel)"
	} else {
		statusBar = "\n/ filter | enter detail | t transition | q quit"
	}

	detailRender := detailPanelStyle.Render(detailContent.String())
	statusBarRender := statusBarStyle.Render(statusBar)

	return detailRender + "\n\n" + statusBarRender
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

func (m model) updateEditDescriptionView(msg tea.Msg) (tea.Model, tea.Cmd) {

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = listView
			m.editingDescription = false
			m.editTextArea.Blur()
			return m, nil
		case "ctrl+s":
			m.mode = listView
			m.editingDescription = false
			m.editTextArea.Blur()
			return m, m.postDescription(m.issueDetail.Key, m.editTextArea.Value())
		}
	}

	var cmd tea.Cmd
	m.editTextArea, cmd = m.editTextArea.Update(msg)

	return m, cmd
}

func (m model) renderEditDescriptionView() string {

	background := m.renderListView()

	var modalContent strings.Builder

	if m.issueDetail != nil {
		header := detailHeaderStyle.Render(m.issueDetail.Key) + " " + renderStatusBadge(m.issueDetail.Status)
		modalContent.WriteString(header + "\n\n")
	}
	modalContent.WriteString("Description:\n")
	modalContent.WriteString(m.editTextArea.Value() + "\n\n")
	modalContent.WriteString("ctrl+s save | esc cancel")

	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("62")).
		Padding(1, 2).
		Width(60).
		Background(lipgloss.Color("150"))

	styledModal := modalStyle.Render(modalContent.String())

	backgroundLayer := lipgloss.NewLayer(background)
	modalLayer := lipgloss.NewLayer(styledModal).X(m.windowWidth / 2).Y(m.windowHeight / 2)

	canvas := lipgloss.NewCanvas(backgroundLayer, modalLayer)
	return canvas.Render()
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

func (m model) postDescription(issueKey, description string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostDescription(context.Background(), issueKey, description)
		if m.client == nil {
			return errMsg{err}
		}

		return editedDescriptionMsg{success: true}
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

func renderStatusBadge(status string) string {
	if status == "selected for development" {
		status = "To Do"
	}

	statusLower := strings.ToLower(status)

	if strings.Contains(statusLower, "trabajando") {
		return statusInProgressStyle.Render(status)
	} else if strings.Contains(statusLower, "done") {
		return statusDoneStyle.Render(status)
	} else if strings.Contains(statusLower, "backlog") || strings.Contains(statusLower, "to do") {
		return statusToDoStyle.Render(status)
	}

	return statusDefaultStyle.Render(status)
}

func renderField(label, value string) string {
	return detailLabelStyle.Render(label+": ") + detailValueStyle.Render(value)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func main() {
	url := os.Getenv("JIRA_URL")
	email := os.Getenv("JIRA_EMAIL")
	token := os.Getenv("JIRA_TOKEN")

	client, _ := jira.NewClient(url, email, token)

	filterBox := textinput.New()
	filterBox.CharLimit = 50

	editTextAreaBox := textarea.New()
	editTextAreaBox.CharLimit = 3000

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
