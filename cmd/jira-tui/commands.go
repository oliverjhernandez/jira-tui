package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

// bubbletea messages from commands
type issuesLoadedMsg struct {
	issues []jira.Issue
}

type epicChildrenLoadedMsg struct {
	children []jira.Issue
}

type myselfLoadedMsg struct {
	me *jira.User
}

type statusesLoadedMsg struct {
	statuses []jira.Status
}

type prioritiesLoadedMsg struct {
	priorities []jira.Priority
}

type issueDetailLoadedMsg struct {
	detail *jira.IssueDetail
}

type transitionsLoadedMsg struct {
	transitions []jira.Transition
}

type assignUsersLoadedMsg struct {
	users []jira.User
}

type workLogsLoadedMsg struct {
	workLogs []jira.WorkLog
}

type worklogTotalsLoadedMsg struct {
	totals map[string]int // issue ID -> total seconds
}

type transitionCompleteMsg struct {
	success bool
}

type linkIssueCompleteMsg struct {
	success bool
}

type editedDescriptionMsg struct {
	success bool
}

type editedPriorityMsg struct {
	success bool
}

type postedCommentMsg struct {
	success bool
}

type postedWorkLog struct {
	success bool
}

type postedEstimateMsg struct {
	success bool
}

type keyTimeoutMsg struct {
	err error
}

type errMsg struct {
	err error
}

func (m model) fetchMySelf() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		me, err := m.client.GetMySelf(context.Background())
		if err != nil {
			return errMsg{err}
		}

		return myselfLoadedMsg{me}
	}
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

func (m model) postTransition(issueKey, transitionID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostTransition(context.Background(), issueKey, transitionID)
		if err != nil {
			return errMsg{err}
		}

		return transitionCompleteMsg{success: true}
	}
}

func (m model) postTransitionWithReason(issueKey, transitionID, reason string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		comment := "Motivo de cancelación: " + reason

		err := m.client.PostTransitionWithComment(context.Background(), issueKey, transitionID, nil, comment)
		if err != nil {
			return errMsg{err}
		}

		return transitionCompleteMsg{success: true}
	}
}

func (m model) postAssignee(issueKey, assigneeID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostAssignee(context.Background(), issueKey, assigneeID)
		if err != nil {
			return errMsg{err}
		}

		return transitionCompleteMsg{success: true}
	}
}

func (m model) updateDescription(issueKey, description string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.UpdateDescription(context.Background(), issueKey, description)
		if err != nil {
			return errMsg{err}
		}

		return editedDescriptionMsg{success: true}
	}
}

func (m model) postPriority(issueKey, priorityName string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.UpdatePriority(context.Background(), issueKey, priorityName)
		if err != nil {
			return errMsg{err}
		}

		return editedPriorityMsg{success: true}
	}
}

func (m model) fetchMyIssues() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		issues, err := m.client.GetMyIssues(context.Background())
		if err != nil {
			return errMsg{err}
		}

		return issuesLoadedMsg{issues}
	}
}

func (m model) fetchEpicChildren(epicKey string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		children, err := m.client.GetEpicChildren(context.Background(), epicKey)
		if err != nil {
			return errMsg{err}
		}

		return epicChildrenLoadedMsg{children}
	}
}

func (m model) fetchPriorities() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		priorities, err := m.client.GetPriorities(context.Background())
		if err != nil {
			return errMsg{err}
		}

		return prioritiesLoadedMsg{priorities}
	}
}

func (m model) fetchStatuses() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		statuses, err := m.client.GetStatuses(context.Background(), Projects)
		if err != nil {
			return errMsg{err}
		}

		return statusesLoadedMsg{statuses}
	}
}

func (m model) postComment(issueKey, comment string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostComment(context.Background(), issueKey, comment, m.usersCache)
		if err != nil {
			return errMsg{err}
		}

		return postedCommentMsg{success: true}
	}
}

func (m model) fetchUsers(issueKey string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		users, err := m.client.GetUsers(context.Background(), issueKey)
		if err != nil {
			return errMsg{err}
		}

		return assignUsersLoadedMsg{users}
	}
}

func (m model) fetchWorkLogs(issueID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		wls, err := m.client.GetWorkLogs(context.Background(), issueID)
		if err != nil {
			return errMsg{err}
		}

		return workLogsLoadedMsg{wls}
	}
}

func (m model) fetchAllWorklogTotals(issues []jira.Issue) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		type result struct {
			issueID string
			total   int
			err     error
		}

		results := make(chan result, len(issues))

		semaphore := make(chan struct{}, 5)

		for _, issue := range issues {
			go func(issueID string) {
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				wls, err := m.client.GetWorkLogs(context.Background(), issueID)
				if err != nil {
					results <- result{issueID: issueID, total: -1, err: err}
					return
				}
				var total int
				for _, wl := range wls {
					total += wl.Time
				}
				results <- result{issueID: issueID, total: total, err: nil}
			}(issue.ID)
		}

		totals := make(map[string]int)
		for range issues {
			r := <-results
			if r.err == nil {
				totals[r.issueID] = r.total
			}
		}

		return worklogTotalsLoadedMsg{totals}
	}
}

func (m model) postWorkLog(issueID, date, accountID string, time int) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostWorkLog(context.Background(), issueID, date, accountID, time)
		if err != nil {
			return errMsg{err}
		}

		return postedWorkLog{success: true}
	}
}

func (m model) postEstimate(issueKey, estimate string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.UpdateOriginalEstimate(context.Background(), issueKey, estimate)
		if err != nil {
			return errMsg{err}
		}

		return postedEstimateMsg{success: true}
	}
}

func (m *model) classifyIssues(issues []jira.Issue, statuses []jira.Status) []Section {

	sections := []Section{
		{Name: "In Progress", CategoryKey: "indeterminate"},
		{Name: "To Do", CategoryKey: "new"},
		{Name: "Done", CategoryKey: "done", Collapsed: true},
	}

	statusCategories := make(map[string]string)
	for _, s := range statuses {
		statusCategories[strings.ToLower(s.Name)] = s.StatusCategory.Key
	}

	for i := range issues {
		issue := &issues[i]
		categoryKey := statusCategories[strings.ToLower(issue.Status)]

		if strings.Contains(strings.ToLower(issue.Status), "validación") {
			categoryKey = "done"
		} // NOTE: probably should find a better way to show validación status

		for idx := range sections {
			if sections[idx].CategoryKey == categoryKey {
				sections[idx].Issues = append(sections[idx].Issues, issue)
				break
			}
		}
	}

	return sections
}

func (m *model) toggleRightColumnView() {
	if m.rightColumnView == worklogsView {
		m.rightColumnView = epicChildrenView
	} else {
		m.rightColumnView = worklogsView
	}
}

func (m model) linkIssue(fromKey, toKey string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostIssueLink(context.Background(), fromKey, toKey)
		if err != nil {
			return errMsg{err}
		}

		return linkIssueCompleteMsg{success: true}
	}
}

func (m model) unlinkIssue(linkID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.DeleteIssueLink(context.Background(), linkID)
		if err != nil {
			return errMsg{err}
		}

		return linkIssueCompleteMsg{success: true}
	}
}

type detailLayout struct {
	leftColumnWidth  int
	rightColumnWidth int
	descHeight       int
	commentsHeight   int
	worklogsHeight   int
	subtasksHeight   int
}

func (m model) calculateDetailLayout() detailLayout {
	panelWidth := ui.GetAvailableWidth(m.windowWidth)
	leftColumnWidth := int(float64(panelWidth)*0.6) - 6
	rightColumnWidth := int(float64(panelWidth)*0.4) - 6

	infoPanel := m.renderInfoPanel(panelWidth)
	infoPanelHeight := lipgloss.Height(infoPanel)

	metadataPanel := m.renderMetadataPanel(leftColumnWidth)
	metadataPanelHeight := lipgloss.Height(metadataPanel)

	statusBar := m.renderDetailStatusBar()
	statusBarHeight := lipgloss.Height(statusBar)

	fixedHeight := infoPanelHeight + metadataPanelHeight + statusBarHeight + 8 // gaps
	freeSpace := m.windowHeight - fixedHeight

	descHeight := freeSpace / 2
	commentsHeight := freeSpace / 2

	worklogsHeight := 0
	subtasksHeight := 0

	log.Printf("commentsHeight calculated: %d", commentsHeight)

	return detailLayout{
		leftColumnWidth,
		rightColumnWidth,
		descHeight,
		commentsHeight,
		worklogsHeight,
		subtasksHeight,
	}

}

func (m model) renderInfoPanel(width int) string {
	panelWidth := ui.GetAvailableWidth(m.windowWidth)

	userName := "loading..."
	if m.myself != nil {
		userName = "@" + m.myself.Name
	}

	var inProgress, toDo, done int
	for _, s := range m.sections {
		switch s.CategoryKey {
		case "indeterminate":
			inProgress = len(s.Issues)
		case "new":
			toDo = len(s.Issues)
		case "done":
			done = len(s.Issues)
		}
	}
	total := inProgress + toDo + done

	projectsStr := strings.Join(Projects, " · ")

	userStyled := ui.InfoPanelUserStyle.Render(userName)
	projectsStyled := ui.InfoPanelProjectStyle.Render(projectsStr)
	line1InnerWidth := width - 6
	line1Gap := line1InnerWidth - lipgloss.Width(userStyled) - lipgloss.Width(projectsStyled)
	if line1Gap < 0 {
		line1Gap = 1
	}
	line1 := userStyled + strings.Repeat(" ", line1Gap) + projectsStyled

	statusCounts := fmt.Sprintf("%s In Progress: %d    %s To Do: %d    %s Done: %d",
		ui.IconInfoInProgress, inProgress,
		ui.IconInfoToDo, toDo,
		ui.IconInfoDone, done)
	totalStr := ui.InfoPanelTotalStyle.Render(fmt.Sprintf("%d issues", total))
	line2Gap := line1InnerWidth - lipgloss.Width(statusCounts) - lipgloss.Width(totalStr)
	if line2Gap < 0 {
		line2Gap = 1
	}
	line2 := statusCounts + strings.Repeat(" ", line2Gap) + totalStr

	var totalLoggedSeconds int
	for _, seconds := range m.worklogTotals {
		totalLoggedSeconds += seconds
	}
	totalLoggedStr := ui.InfoPanelCountLabelStyle.Render(ui.IconTime + " Total Logged: " + ui.FormatTimeSpent(totalLoggedSeconds))
	line3 := totalLoggedStr

	content := line1 + "\n" + line2 + "\n" + line3
	return ui.InfoPanelStyle.Width(panelWidth).Render(content)
}

func (m model) renderMetadataPanel(width int) string {
	index := ui.StatusBarDescStyle.Render(
		fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.sections[m.sectionCursor].Issues)),
	)
	var parent string
	if m.issueDetail.Parent != nil {
		parent = ui.RenderIssueType(m.issueDetail.Parent.Type, false) + " " +
			ui.StatusBarDescStyle.Render(m.issueDetail.Parent.Key+" / ")
	}

	issueKey := ui.RenderIssueType(m.issueDetail.Type, false) + " " + ui.DetailHeaderStyle.Render(m.issueDetail.Key)
	summaryMaxWidth := 50
	issueSummary := ui.DetailValueStyle.Render(truncateLongString(m.issueDetail.Summary, summaryMaxWidth))
	var linkedIssue string
	if m.issueDetail.IsLinkedToChange {
		linkedIssue = "🔗 " + jira.MonthlyChangeIssue
	}

	detailsHeaderLine1 := index + " " + parent + issueKey + "  " + issueSummary + " " + linkedIssue

	status := ui.RenderStatusBadge(m.issueDetail.Status)
	assignee := ui.StatusBarDescStyle.Render("@" + strings.ToLower(strings.Split(m.issueDetail.Assignee, " ")[0]))

	logged := ""
	if m.selectedIssueWorklogs != nil {
		logged = ui.StatusBarDescStyle.Render("Logged: " + extractLoggedTime(m.selectedIssueWorklogs))
	}

	detailsHeaderLine2 := status + "  " + assignee + "  " + logged
	leftHeader := detailsHeaderLine1 + "\n" + detailsHeaderLine2

	colwidth := 30

	col1 := ui.RenderFieldStyled("Priority", ui.RenderPriority(m.issueDetail.Priority.Name, true), colwidth)
	col2 := ui.RenderFieldStyled("Reporter", m.issueDetail.Reporter, colwidth)
	col3 := ui.RenderFieldStyled("Type", ui.RenderIssueType(m.issueDetail.Type, true), colwidth)
	metadataRow1 := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	col4 := ui.RenderFieldStyled("Created", timeAgo(m.issueDetail.Created), colwidth)
	col5 := ui.RenderFieldStyled("Updated", timeAgo(m.issueDetail.Updated), colwidth)
	metadataRow2 := lipgloss.JoinHorizontal(lipgloss.Top, col4, col5)

	metadataRow := metadataRow1 + "\n" + metadataRow2

	var detailsContent strings.Builder
	detailsContent.WriteString(leftHeader + "\n\n")
	detailsContent.WriteString(metadataRow + "\n\n")

	metadataPanel := ui.PanelStyleActive.
		Width(width).
		Render(detailsContent.String())

	return metadataPanel
}

func (m model) renderDetailStatusBar() string {
	var statusBar strings.Builder
	if m.statusMessage != "" {
		statusBar.WriteString(m.statusMessage)
	} else if m.loadingDetail || m.loadingTransitions {
		statusBar.WriteString(m.spinner.View() + "Loading...")
	} else {
		statusBar.WriteString(strings.Join([]string{
			ui.RenderKeyBind("j/k", "scroll"),
			ui.RenderKeyBind("d", "description"),
			ui.RenderKeyBind("p", "priority"),
			ui.RenderKeyBind("c", "comment"),
			ui.RenderKeyBind("w", "worklog"),
			ui.RenderKeyBind("a", "assignee"),
			ui.RenderKeyBind("t", "transition"),
			ui.RenderKeyBind("esc", "back"),
			ui.RenderKeyBind("q", "quit"),
		}, "  "))
	}

	return ui.StatusBarStyle.Render(statusBar.String())
}

func (m model) buildDescriptionContent(width int) string {
	var content strings.Builder
	// content.WriteString(ui.SeparatorStyle.Render(strings.Repeat("─", 4)+" ") +
	// 	ui.SectionTitleStyle.Render("󰠮 Description ") +
	// 	ui.SeparatorStyle.Render(strings.Repeat("─", 20)) + "\n\n")

	if m.issueDetail.Description != "" {
		wrappedDesc := ui.DetailValueStyle.Width(width - 4).Render(m.issueDetail.Description)
		content.WriteString(wrappedDesc + "\n\n")
	} else {
		// NOTE: use proper style here :point_down
		content.WriteString(ui.StatusBarDescStyle.Render("No description") + "\n\n")
	}

	return content.String()
}

func (m model) renderDescriptionPanel(width, height int) string {
	viewport := m.descViewport.View()

	var style lipgloss.Style
	if m.focusedSection == descSection {
		style = ui.PanelActiveStyle
	} else {
		style = ui.PanelInactiveStyle
	}

	return style.Width(width).Render(viewport)
}

func (m model) buildCommentsContent(width int) string {
	var content strings.Builder
	commentCount := len(m.issueDetail.Comments)
	// content.WriteString(ui.SeparatorStyle.Render(strings.Repeat("─", 4)+" ") +
	// 	ui.SectionTitleStyle.Render(fmt.Sprintf("󱅰 Comments (%d) ", commentCount)) +
	// 	ui.SeparatorStyle.Render(strings.Repeat("─", 60)) + "\n\n")

	if commentCount > 0 {
		for i, c := range m.issueDetail.Comments {
			author := ui.CommentAuthorStyle.Render(c.Author)
			timestamp := ui.CommentTimestampStyle.Render(" • " + timeAgo(c.Created))
			content.WriteString(author + timestamp + "\n")
			wrappedBody := ui.CommentBodyStyle.Width(width - 4).Render(c.Body)
			content.WriteString(wrappedBody + "\n")

			if i < commentCount-1 {
				content.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
			} else {
				content.WriteString("\n")
			}
		}
	}

	return content.String()
}

func (m model) renderCommentsPanel(width, height int) string {
	log.Printf("renderCommentsPanel called with width=%d, height=%d", width, height)
	log.Printf("commentsViewport actual: Width=%d, Height=%d", m.commentsViewport.Width, m.commentsViewport.Height)

	viewport := m.commentsViewport.View()
	log.Printf("viewport.View() returned height: %d", lipgloss.Height(viewport))

	var style lipgloss.Style
	if m.focusedSection == commentsSection {
		style = ui.PanelActiveStyle
	} else {
		style = ui.PanelInactiveStyle
	}

	return style.Width(width).Render(viewport)
}

// func (m model) renderWorklogsPanel(width, height int) string // future
// func (m model) renderSubtasksPanel(width, height int) string // future
