package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

// bubbletea messages from commands
type issuesLoadedMsg struct {
	issues []jira.Issue
}

type childrenLoadedMsg struct {
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

type projectsLoadedMsg struct {
	projects []jira.Project
}

type issueTypesLoadedMsg struct {
	issueTypes []jira.IssueType
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

type newIssueCompleteMsg struct {
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

type updatedCommentMsg struct {
	success bool
}

type deletedCommentMsg struct {
	success bool
}

type postedWorkLog struct {
	success bool
}

type postedEstimateMsg struct {
	success bool
}

type keyTimeoutMsg struct{}

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

func (m model) fetchProjects() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		detail, err := m.client.GetProjects(context.Background())
		if err != nil {
			return errMsg{err}
		}

		var projects []jira.Project
		for _, jp := range detail {
			project := jira.Project{
				ID:   jp.ID,
				Name: jp.Name,
				Key:  jp.Key,
			}

			projects = append(projects, project)
		}

		return projectsLoadedMsg{projects}
	}
}

func (m model) fetchIssueTypes() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		detail, err := m.client.GetIssueTypes(context.Background())
		if err != nil {
			return errMsg{err}
		}

		return issueTypesLoadedMsg{detail}
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

func (m model) postNewIssue(issue *NewIssueFormData) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		var projectID string
		for _, p := range m.projects {
			log.Printf("Project: ID %s - Name %s", p.ID, p.Key)
			if strings.Contains(p.Key, issue.ProjectName) {
				projectID = p.ID
				break
			} else {
				projectID = ""
			}
		}

		var issueTypeID string
		for _, it := range m.issueTypes {
			if it.Name == issue.IssueTypeName {
				if it.Scope != nil {
				}
				if it.Scope != nil && it.Scope.Project.ID == projectID {
					issueTypeID = it.ID
					break
				} else {
					issueTypeID = ""
				}
			}
		}

		if issueTypeID == "" {
			for _, it := range m.issueTypes {
				if it.Name == issue.IssueTypeName && it.Scope == nil {
					issueTypeID = it.ID
					break
				}
			}
		}

		// TODO: validate estimate
		originalEstimate := issue.OriginalEstimate
		summary := issue.Summary

		var assigneeID string
		if issue.AssigneeName == m.myself.Name {
			assigneeID = m.myself.ID
		}

		var priorityID string
		for _, p := range m.priorities {
			if strings.Contains(p.Name, issue.PriorityName) {
				priorityID = p.ID
				break
			}
		}

		var dueDate time.Time
		if _, err := time.Parse("2006-01-02", issue.DueDate); err != nil {
			dueDate = time.Now()
		}

		description := buildSimpleDescriptionContent(issue.Description)

		err := m.client.PostNewIssue(
			context.Background(),
			projectID,
			issueTypeID,
			originalEstimate,
			summary,
			assigneeID,
			priorityID,
			description,
			dueDate.Format("2006-01-02"),
		)
		if err != nil {
			return errMsg{err}
		}

		return newIssueCompleteMsg{success: true}
	}
}

func (m model) postTransition(issueKey, transitionID, transitionName string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		var workLogTime string
		if transitionName == "Done" {
			workLogTime = extractLoggedTime(m.selectedIssueWorklogs)
		}

		err := m.client.PostTransition(context.Background(), issueKey, transitionID, nil, "", workLogTime)
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

		err := m.client.PostTransition(context.Background(), issueKey, transitionID, nil, comment, "")
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

		children, err := m.client.GetChildren(context.Background(), epicKey)
		if err != nil {
			return errMsg{err}
		}

		return childrenLoadedMsg{children}
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

		var projects []string
		for _, p := range m.activeProjects {
			projects = append(projects, p.ID)
		}

		statuses, err := m.client.GetStatuses(context.Background(), projects)
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

func (m model) updateComment(issueKey, commentID, comment string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PutComment(
			context.Background(),
			issueKey,
			commentID,
			comment,
			m.usersCache,
		)
		if err != nil {
			return errMsg{err}
		}

		return updatedCommentMsg{success: true}
	}
}

func (m model) deleteComment(issueKey, commentID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.DeleteComment(
			context.Background(),
			issueKey,
			commentID,
		)
		if err != nil {
			return errMsg{err}
		}

		return deletedCommentMsg{success: true}
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
	childrenHeight   int
}

func (m model) calculateDetailLayout() detailLayout {
	panelWidth := ui.GetAvailableWidth(m.windowWidth)
	leftColumnWidth := int(float64(panelWidth) * 0.8)
	rightColumnWidth := int(float64(panelWidth)*0.2) - 1

	metadataPanel := m.renderMetadataPanel(leftColumnWidth)
	metadataPanelHeight := lipgloss.Height(metadataPanel)

	statusBar := m.renderDetailStatusBar()
	statusBarHeight := lipgloss.Height(statusBar)

	leftFixedHeight := metadataPanelHeight + statusBarHeight + 8 // gaps
	rightFixedHeight := statusBarHeight + 8
	leftColumnFreeHeight := m.windowHeight - leftFixedHeight
	rightColumnFreeHeight := m.windowHeight - rightFixedHeight

	descHeight := leftColumnFreeHeight / 2
	commentsHeight := leftColumnFreeHeight / 2

	worklogsHeight := rightColumnFreeHeight/2 - 1
	childrenHeight := rightColumnFreeHeight / 2

	return detailLayout{
		leftColumnWidth,
		rightColumnWidth,
		descHeight,
		commentsHeight,
		worklogsHeight,
		childrenHeight,
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

	var projectsStr string
	if m.activeProjects != nil {
		var projects []string
		for _, p := range m.activeProjects {
			projects = append(projects, p.Name)
		}

		projectsStr = strings.Join(projects, " · ")
	}

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

	metadataRows := metadataRow1 + "\n" + metadataRow2

	var detailsContent strings.Builder
	detailsContent.WriteString(leftHeader + "\n")
	detailsContent.WriteString(metadataRows)

	var style lipgloss.Style
	if m.focusedSection == metadataSection {
		style = ui.PanelActiveStyle
	} else {
		style = ui.PanelActiveSecondaryStyle
	}

	metadataPanel := style.
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

	if m.issueDetail.Description != nil {
		descText := jira.ExtractText(m.issueDetail.Description, width-4)
		wrappedDesc := ui.DetailValueStyle.Width(width - 4).Render(descText)
		content.WriteString(wrappedDesc + "\n\n")
	} else {
		content.WriteString(ui.StatusBarDescStyle.Render("No description") + "\n\n")
	}
	return content.String()
}

func (m model) renderDescriptionPanel(width int) string {
	viewport := m.descViewport.View()

	var style lipgloss.Style
	if m.focusedSection == descriptionSection {
		style = ui.PanelActiveStyle
	} else {
		style = ui.PanelInactiveStyle
	}

	return style.Width(width).Render(viewport)
}

func (m model) buildCommentsContent(width int) string {
	var content strings.Builder
	comments := m.issueDetail.Comments
	commentCount := len(comments)

	if commentCount > 0 {
		for i, c := range comments {
			isSelected := m.commentsCursor == i
			isLast := i == commentCount-1

			commentStr := m.renderComment(c, width, isSelected, isLast)
			content.WriteString(commentStr)
		}
	}

	return content.String()
}

func (m model) renderComment(c jira.Comment, width int, isSelected bool, isLast bool) string {
	var comment strings.Builder

	author := ui.CommentAuthorStyle.Render(c.Author)
	timestamp := ui.CommentTimestampStyle.Render(" • " + timeAgo(c.Created))

	if c.Updated != c.Created {
		timestamp += ui.StatusBarDescStyle.Render(" (edited)")
	}

	if isSelected {
		cursor := ui.IconCursor
		comment.WriteString(cursor + ui.SelectedRowStyle.Render(author+timestamp) + "\n")
	} else {
		comment.WriteString(author + timestamp + "\n")
	}

	bodyText := jira.ExtractText(c.Body, width-4)
	wrappedBody := ui.CommentBodyStyle.Width(width - 4).Render(bodyText)
	comment.WriteString(wrappedBody + "\n")

	if !isLast {
		comment.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
	} else {
		comment.WriteString("\n")
	}

	return comment.String()
}

func (m model) renderCommentsPanel(width int) string {
	viewport := m.commentsViewport.View()

	var style lipgloss.Style
	if m.focusedSection == commentsSection {
		style = ui.PanelActiveStyle
	} else {
		style = ui.PanelInactiveStyle
	}

	return style.Width(width).Render(viewport)
}

func (m model) buildWorklogsContent(width int) string {
	var content strings.Builder

	if m.issueDetail.Comments != nil && len(m.issueDetail.Comments) > 0 {
		for i, c := range m.selectedIssueWorklogs {
			user := m.getUserName(c.Author.AccountID)
			time := ui.WorklogsAuthorStyle.Render(strconv.Itoa(c.Time))
			author := ui.WorklogsAuthorStyle.Render(user)
			timestamp := ui.CommentTimestampStyle.Render(" • " + timeAgo(c.UpdatedAt))
			content.WriteString(time + author + timestamp + "\n")
			description := ui.CommentBodyStyle.Width(width - 4).Render(c.Description)
			content.WriteString(description + "\n")

			if i < len(m.issueDetail.Comments)-1 {
				content.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
			} else {
				content.WriteString("\n")
			}
		}
	}

	return content.String()
}

func (m model) renderWorklogsPanel(width int) string {
	viewport := m.worklogsViewport.View()

	var style lipgloss.Style
	if m.focusedSection == worklogsSection {
		style = ui.PanelActiveStyle
	} else {
		style = ui.PanelInactiveStyle
	}

	return style.Width(width).Render(viewport)
}

func (m model) getUserName(accountID string) string {
	for _, u := range m.usersCache {
		if u.ID == accountID {
			return u.Name
		}
	}

	return accountID
}

func (m model) buildChildrenContent(width int) string {
	var content strings.Builder
	sortIssuesByStatus(m.issueDetail.Children)
	childrenCount := len(m.issueDetail.Children)

	if childrenCount > 0 {
		for i, c := range m.issueDetail.Children {
			issue := ui.RenderIssueType(c.Type, false)
			key := c.Key
			priority := ui.RenderPriority(c.Priority, false)
			status := ui.RenderStatusBadge(c.Status)
			assignee := ui.StatusBarDescStyle.Render("@" + strings.ToLower(strings.Split(c.Assignee, " ")[0]))

			content.WriteString(issue + " " + key + " " + priority + " " + status + "\n")
			content.WriteString(assignee + "\n")
			summary := ui.CommentBodyStyle.Width(width - 4).Render(c.Summary)
			content.WriteString(summary + "\n")

			if i < childrenCount-1 {
				content.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
			} else {
				content.WriteString("\n")
			}
		}
	}

	return content.String()
}

func (m model) renderChildrenPanel(width int) string {
	viewport := m.childrenViewport.View()

	var style lipgloss.Style
	if m.focusedSection == childrenSection {
		style = ui.PanelActiveStyle
	} else {
		style = ui.PanelInactiveStyle
	}

	return style.Width(width).Render(viewport)
}

func (m model) renderSimpleBackground() string {
	bg := lipgloss.NewStyle().
		Width(m.windowWidth).
		Height(m.windowHeight).
		Background(lipgloss.Color("0")).
		Render("")

	return bg
}

// {
//   "issues": [
//     {
//       "expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
//       "id": "54792",
//       "self": "https://layer7.atlassian.net/rest/api/3/issue/54792",
//       "key": "DEV-1469",
//       "fields": {
//         "summary": "CCC4, UPDATE al Script de Grabaciones, no se están copiando completamente desde TOTP a S3",
//         "issuetype": {
//           "self": "https://layer7.atlassian.net/rest/api/3/issuetype/10012",
//           "id": "10012",
//           "description": "A small, distinct piece of work.",
//           "iconUrl": "https://layer7.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10318?size=medium",
//           "name": "Task",
//           "subtask": false,
//           "avatarId": 10318,
//           "hierarchyLevel": 0
//         },
//         "assignee": {
//           "self": "https://layer7.atlassian.net/rest/api/3/user?accountId=712020%3Ad1924ca8-be15-49b1-9484-defcf52776b2",
//           "accountId": "712020:d1924ca8-be15-49b1-9484-defcf52776b2",
//           "emailAddress": "oliver@layer7.mx",
//           "avatarUrls": {
//             "48x48": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "24x24": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "16x16": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "32x32": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png"
//           },
//           "displayName": "Oliver Hernández",
//           "active": true,
//           "timeZone": "America/Caracas",
//           "accountType": "atlassian"
//         },
//         "priority": {
//           "self": "https://layer7.atlassian.net/rest/api/3/priority/1",
//           "iconUrl": "https://layer7.atlassian.net/rest/api/2/universal_avatar/view/type/priority/avatar/10781?size=medium",
//           "name": "Highest",
//           "id": "1"
//         },
//         "status": {
//           "self": "https://layer7.atlassian.net/rest/api/3/status/10013",
//           "description": "",
//           "iconUrl": "https://layer7.atlassian.net/",
//           "name": "Done",
//           "id": "10013",
//           "statusCategory": {
//             "self": "https://layer7.atlassian.net/rest/api/3/statuscategory/3",
//             "id": 3,
//             "key": "done",
//             "colorName": "green",
//             "name": "Done"
//           }
//         }
//       }
//     },
//     {
//       "expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
//       "id": "54791",
//       "self": "https://layer7.atlassian.net/rest/api/3/issue/54791",
//       "key": "DEV-1468",
//       "fields": {
//         "summary": "CCC3, UPDATE al Script de Grabaciones, no se están copiando completamente desde TOTP a S3",
//         "issuetype": {
//           "self": "https://layer7.atlassian.net/rest/api/3/issuetype/10012",
//           "id": "10012",
//           "description": "A small, distinct piece of work.",
//           "iconUrl": "https://layer7.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10318?size=medium",
//           "name": "Task",
//           "subtask": false,
//           "avatarId": 10318,
//           "hierarchyLevel": 0
//         },
//         "assignee": {
//           "self": "https://layer7.atlassian.net/rest/api/3/user?accountId=712020%3Ad1924ca8-be15-49b1-9484-defcf52776b2",
//           "accountId": "712020:d1924ca8-be15-49b1-9484-defcf52776b2",
//           "emailAddress": "oliver@layer7.mx",
//           "avatarUrls": {
//             "48x48": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "24x24": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "16x16": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "32x32": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png"
//           },
//           "displayName": "Oliver Hernández",
//           "active": true,
//           "timeZone": "America/Caracas",
//           "accountType": "atlassian"
//         },
//         "priority": {
//           "self": "https://layer7.atlassian.net/rest/api/3/priority/3",
//           "iconUrl": "https://layer7.atlassian.net/rest/api/2/universal_avatar/view/type/priority/avatar/10812?size=medium",
//           "name": "Medium",
//           "id": "3"
//         },
//         "status": {
//           "self": "https://layer7.atlassian.net/rest/api/3/status/10013",
//           "description": "",
//           "iconUrl": "https://layer7.atlassian.net/",
//           "name": "Done",
//           "id": "10013",
//           "statusCategory": {
//             "self": "https://layer7.atlassian.net/rest/api/3/statuscategory/3",
//             "id": 3,
//             "key": "done",
//             "colorName": "green",
//             "name": "Done"
//           }
//         }
//       }
//     },
//     {
//       "expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
//       "id": "54790",
//       "self": "https://layer7.atlassian.net/rest/api/3/issue/54790",
//       "key": "DEV-1467",
//       "fields": {
//         "summary": "CCC1, UPDATE al Script de Grabaciones, no se están copiando completamente desde TOTP a S3",
//         "issuetype": {
//           "self": "https://layer7.atlassian.net/rest/api/3/issuetype/10012",
//           "id": "10012",
//           "description": "A small, distinct piece of work.",
//           "iconUrl": "https://layer7.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10318?size=medium",
//           "name": "Task",
//           "subtask": false,
//           "avatarId": 10318,
//           "hierarchyLevel": 0
//         },
//         "assignee": {
//           "self": "https://layer7.atlassian.net/rest/api/3/user?accountId=712020%3Ad1924ca8-be15-49b1-9484-defcf52776b2",
//           "accountId": "712020:d1924ca8-be15-49b1-9484-defcf52776b2",
//           "emailAddress": "oliver@layer7.mx",
//           "avatarUrls": {
//             "48x48": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "24x24": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "16x16": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "32x32": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png"
//           },
//           "displayName": "Oliver Hernández",
//           "active": true,
//           "timeZone": "America/Caracas",
//           "accountType": "atlassian"
//         },
//         "priority": {
//           "self": "https://layer7.atlassian.net/rest/api/3/priority/1",
//           "iconUrl": "https://layer7.atlassian.net/rest/api/2/universal_avatar/view/type/priority/avatar/10781?size=medium",
//           "name": "Highest",
//           "id": "1"
//         },
//         "status": {
//           "self": "https://layer7.atlassian.net/rest/api/3/status/10013",
//           "description": "",
//           "iconUrl": "https://layer7.atlassian.net/",
//           "name": "Done",
//           "id": "10013",
//           "statusCategory": {
//             "self": "https://layer7.atlassian.net/rest/api/3/statuscategory/3",
//             "id": 3,
//             "key": "done",
//             "colorName": "green",
//             "name": "Done"
//           }
//         }
//       }
//     },
//     {
//       "expand": "renderedFields,names,schema,operations,editmeta,changelog,versionedRepresentations",
//       "id": "53877",
//       "self": "https://layer7.atlassian.net/rest/api/3/issue/53877",
//       "key": "DEV-1441",
//       "fields": {
//         "summary": "CCC2 - Grabaciones no se están copiando completamente desde TOTP a S3",
//         "issuetype": {
//           "self": "https://layer7.atlassian.net/rest/api/3/issuetype/10012",
//           "id": "10012",
//           "description": "A small, distinct piece of work.",
//           "iconUrl": "https://layer7.atlassian.net/rest/api/2/universal_avatar/view/type/issuetype/avatar/10318?size=medium",
//           "name": "Task",
//           "subtask": false,
//           "avatarId": 10318,
//           "hierarchyLevel": 0
//         },
//         "assignee": {
//           "self": "https://layer7.atlassian.net/rest/api/3/user?accountId=712020%3Ad1924ca8-be15-49b1-9484-defcf52776b2",
//           "accountId": "712020:d1924ca8-be15-49b1-9484-defcf52776b2",
//           "emailAddress": "oliver@layer7.mx",
//           "avatarUrls": {
//             "48x48": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "24x24": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "16x16": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png",
//             "32x32": "https://secure.gravatar.com/avatar/8b4900d6b6d1b0b66b20fc6feb3d2e70?d=https%3A%2F%2Favatar-management--avatars.us-west-2.prod.public.atl-paas.net%2Finitials%2FOH-5.png"
//           },
//           "displayName": "Oliver Hernández",
//           "active": true,
//           "timeZone": "America/Caracas",
//           "accountType": "atlassian"
//         },
//         "priority": {
//           "self": "https://layer7.atlassian.net/rest/api/3/priority/1",
//           "iconUrl": "https://layer7.atlassian.net/rest/api/2/universal_avatar/view/type/priority/avatar/10781?size=medium",
//           "name": "Highest",
//           "id": "1"
//         },
//         "status": {
//           "self": "https://layer7.atlassian.net/rest/api/3/status/10013",
//           "description": "",
//           "iconUrl": "https://layer7.atlassian.net/",
//           "name": "Done",
//           "id": "10013",
//           "statusCategory": {
//             "self": "https://layer7.atlassian.net/rest/api/3/statuscategory/3",
//             "id": 3,
//             "key": "done",
//             "colorName": "green",
//             "name": "Done"
//           }
//         }
//       }
//     }
//   ],
//   "isLast": true
// }
