package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type detailLayout struct {
	leftColumnWidth  int
	rightColumnWidth int
	descHeight       int
	commentsHeight   int
	worklogsHeight   int
	subTasksHeight   int
}

type listLayout struct {
	panelContentWidth int
	infoHeight        int
	listHeight        int
	statusBarHeight   int
}

// bubbletea messages from commands
type issuesLoadedMsg struct {
	issues []jira.Issue
}

type subTasksLoadedMsg struct {
	subTasks []jira.Issue
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

type assignableUsersLoadedMsg struct {
	users []jira.User
}

type usersLoadedMsg struct {
	users []jira.User
}

type workLogsLoadedMsg struct {
	workLogs []jira.Worklog
}

type worklogTotalsLoadedMsg struct {
	totals map[string]int // issue ID -> total seconds
}

type transitionPostedMsg struct{}

type assigneePostedMsg struct{}

type newIssuePostedMsg struct{}

type issueLinkPostedMsg struct{}

type updatedDescriptionMsg struct{}

type priorityPostedMsg struct{}

type commentPostedMsg struct{}

type commentUpdatedMsg struct{}

type commentDeletedMsg struct{}

type workLogPostedMsg struct{}

type workLogUpdatedMsg struct{}

type workLogDeletedMsg struct{}

type estimatePostedMsg struct{}

type keyTimeoutMsg struct{}

type clearStatusMsg struct{}

type myIssuesPollMsg struct{}

type issueDetailPollMsg struct{}

type errMsg struct {
	err error
}

func (m model) fetchMySelfCmd() tea.Cmd {
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

func (m model) fetchIssueDetailCmd(issueKey string) tea.Cmd {
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

func (m model) fetchProjectsCmd() tea.Cmd {
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

func (m model) fetchIssueTypesCmd() tea.Cmd {
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

func (m model) fetchTransitionsCmd(issueKey string) tea.Cmd {
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

func (m model) postNewIssueCmd(data *NewIssueFormData) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		var projectID string
		for _, p := range m.projects {
			if strings.Contains(p.Name, data.ProjectName) {
				projectID = p.ID
				break
			} else {
				projectID = ""
			}
		}

		var issueTypeID string
		for _, it := range m.issueTypes {
			if it.Name == data.IssueTypeName {
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
				if it.Name == data.IssueTypeName && it.Scope == nil {
					issueTypeID = it.ID
					break
				}
			}
		}

		// TODO: validate estimate
		originalEstimate := data.OriginalEstimate
		parentKey := data.ParentKey
		summary := data.Summary

		var assigneeID string
		if data.AssigneeName == m.myself.Name {
			assigneeID = m.myself.ID
		}

		var priorityID string
		for _, p := range m.priorities {
			if strings.Contains(p.Name, data.PriorityName) {
				priorityID = p.ID
				break
			}
		}

		var dueDate string
		if _, err := time.Parse("2006-01-02", data.DueDate); err != nil {
			dueDate = time.Now().Format("2006-01-02")
		} else {
			dueDate = data.DueDate
		}

		description := buildSimpleDescriptionContent(data.Description)

		err := m.client.PostNewIssue(
			context.Background(),
			projectID,
			issueTypeID,
			originalEstimate,
			summary,
			parentKey,
			assigneeID,
			priorityID,
			description,
			dueDate,
		)
		if err != nil {
			return errMsg{err}
		}

		return newIssuePostedMsg{}
	}
}

func (m model) postTransitionCmd(issueKey, transitionID, transitionName string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		var workLogTime string
		if (transitionName == "Done" || transitionName == "Validación") && m.activeIssue.Type == "Task" {
			workLogTime = "1m"
		}

		err := m.client.PostTransition(context.Background(), issueKey, transitionID, nil, "", workLogTime)
		if err != nil {
			return errMsg{err}
		}
		return transitionPostedMsg{}
	}
}

func (m model) postTransitionWithReasonCmd(issueKey, transitionID, reason string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		comment := "Motivo de cancelación: " + reason

		err := m.client.PostTransition(context.Background(), issueKey, transitionID, nil, comment, "")
		if err != nil {
			return errMsg{err}
		}

		return transitionPostedMsg{}
	}
}

func (m model) postAssigneeCmd(issueKey, assigneeID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostAssignee(context.Background(), issueKey, assigneeID)
		if err != nil {
			return errMsg{err}
		}

		return assigneePostedMsg{}
	}
}

func (m model) updateDescriptionCmd(issueKey, description string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.UpdateDescription(context.Background(), issueKey, description)
		if err != nil {
			return errMsg{err}
		}

		return updatedDescriptionMsg{}
	}
}

func (m model) postPriorityCmd(issueKey, priorityName string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.UpdatePriority(context.Background(), issueKey, priorityName)
		if err != nil {
			return errMsg{err}
		}

		return priorityPostedMsg{}
	}
}

func (m model) fetchMyIssuesCmd() tea.Cmd {
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

func (m model) fetchSubTasksCmd(parentKey string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		subTasks, err := m.client.GetSubTasks(context.Background(), parentKey)
		if err != nil {
			return errMsg{err}
		}

		return subTasksLoadedMsg{subTasks}
	}
}

func (m model) fetchPrioritiesCmd() tea.Cmd {
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

func (m model) fetchStatusesCmd(projects []jira.Project) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		statuses, err := m.client.GetStatuses(context.Background(), projects)
		if err != nil {
			return errMsg{err}
		}

		return statusesLoadedMsg{statuses}
	}
}

func (m model) postCommentCmd(issueKey, comment string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostComment(context.Background(), issueKey, comment, m.usersCache)
		if err != nil {
			return errMsg{err}
		}

		return commentPostedMsg{}
	}
}

func (m model) updateCommentCmd(issueKey, commentID, comment string) tea.Cmd {
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

		return commentUpdatedMsg{}
	}
}

func (m model) deleteCommentCmd(issueKey, commentID string) tea.Cmd {
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

		return commentDeletedMsg{}
	}
}

func (m model) fetchAssignableUsersCmd(issueKey string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		users, err := m.client.GetAssignableUsers(context.Background(), issueKey)
		if err != nil {
			return errMsg{err}
		}

		return assignableUsersLoadedMsg{users}
	}
}

func (m model) fetchAllUsersCmd() tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		users, err := m.client.GetAllUsers(context.Background())
		if err != nil {
			return errMsg{err}
		}

		return usersLoadedMsg{users}
	}
}

func (m model) fetchWorkLogsCmd(issueID string) tea.Cmd {
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

func (m model) fetchAllWorklogsTotalCmd(issues []jira.Issue) tea.Cmd {
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

func (m model) postWorkLogCmd(issueID, startDate, accountID, description string, time int) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostWorkLog(
			context.Background(),
			issueID,
			startDate,
			accountID,
			description,
			time,
		)
		if err != nil {
			return errMsg{err}
		}

		return workLogPostedMsg{}
	}
}

func (m model) putWorkLogCmd(worklogID, issueID, startDate, accountID, description string, time int) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PutWorkLog(context.Background(), worklogID, issueID, startDate, accountID, description, time)
		if err != nil {
			return errMsg{err}
		}

		return workLogUpdatedMsg{}
	}
}

func (m model) deleteWorkLogCmd(worklogID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.DeleteWorkLog(context.Background(), worklogID)
		if err != nil {
			return errMsg{err}
		}

		return workLogDeletedMsg{}
	}
}

func (m model) postEstimateCmd(issueKey, estimate string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.UpdateOriginalEstimate(context.Background(), issueKey, estimate)
		if err != nil {
			return errMsg{err}
		}

		return estimatePostedMsg{}
	}
}

func (m model) linkIssueCmd(fromKey, toKey string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostIssueLink(context.Background(), fromKey, toKey)
		if err != nil {
			return errMsg{err}
		}

		return issueLinkPostedMsg{}
	}
}

func (m model) unlinkIssueCmd(linkID string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.DeleteIssueLink(context.Background(), linkID)
		if err != nil {
			return errMsg{err}
		}

		return issueLinkPostedMsg{}
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

func (m model) calculateDetailLayout() detailLayout {
	panelWidth := m.windowWidth
	leftColumnWidth := int(float64(panelWidth) * 0.8)
	rightColumnWidth := int(float64(panelWidth) * 0.2)

	metadataPanelHeight := 8
	statusBarHeight := 1

	leftFixedHeight := metadataPanelHeight + statusBarHeight + (ui.PanelOverheadHeight * 2)
	rightFixedHeight := statusBarHeight + (ui.PanelOverheadHeight * 2)
	leftColumnFreeHeight := m.windowHeight - leftFixedHeight
	rightColumnFreeHeight := m.windowHeight - rightFixedHeight

	descHeight := leftColumnFreeHeight / 2
	commentsHeight := leftColumnFreeHeight / 2

	worklogsHeight := rightColumnFreeHeight / 2
	subTasksHeight := rightColumnFreeHeight / 2

	return detailLayout{
		leftColumnWidth,
		rightColumnWidth,
		descHeight,
		commentsHeight,
		worklogsHeight,
		subTasksHeight,
	}
}

func (m model) calculateListLayout() listLayout {
	infoHeight := 5
	statusBarHeight := 1
	listHeight := m.windowHeight - infoHeight - statusBarHeight - ui.PanelOverheadHeight
	panelWidth := m.windowWidth - ui.PanelOverheadWidth

	return listLayout{
		panelWidth,
		infoHeight,
		listHeight,
		statusBarHeight,
	}
}

func (m model) buildListContent() string {
	var listContent strings.Builder

	sectionsToRender := m.sections
	if m.filteredSections != nil {
		sectionsToRender = m.filteredSections
	}

	for si, s := range sectionsToRender {
		sectionHeader := ui.SectionTitleStyle.Render(fmt.Sprintf("%s (%d)", s.Name, len(s.Issues)))
		fmt.Fprintf(&listContent, "%s\n", sectionHeader)

		for ii, issue := range s.Issues {
			issueType := ui.RenderIssueType(issue.Type, false)
			key := m.columnWidths.RenderKey(issue.Key)
			priority := ui.RenderPriority(issue.Priority, false)
			summary := m.columnWidths.RenderSummary(truncateLongString(issue.Summary, m.columnWidths.Summary))
			reporter := m.columnWidths.RenderReporter("@" + truncateLongString(issue.Reporter.ID, m.columnWidths.Assignee))
			statusBadge := ui.RenderStatusBadge(issue.Status)
			assignee := m.columnWidths.RenderAssignee("@" + truncateLongString(issue.Assignee, m.columnWidths.Assignee))
			worklogSeconds := m.worklogTotals[issue.ID]
			timeSpent := m.columnWidths.RenderTimeSpent(ui.FormatTimeSpent(worklogSeconds))

			emptySpace := m.columnWidths.RenderEmptySpace()
			line := issueType + emptySpace +
				key +
				priority + emptySpace +
				summary + emptySpace +
				reporter + emptySpace +
				statusBadge + emptySpace +
				assignee + emptySpace +
				timeSpent

			if m.sectionCursor == si && m.cursor == ii {
				cursor := ui.IconCursor
				line = cursor + ui.SelectedRowStyle.Render(line)
			} else {
				line = "  " + ui.NormalRowStyle.Render(line)
			}

			listContent.WriteString(line + "\n")
		}

		listContent.WriteString("\n\n")
	}

	return listContent.String()
}

func (m model) renderInfoPanel() string {
	m.statusMessage = statusMessage{
		msgType: infoStatusBarMsg,
		content: "Loading...",
	}

	var userName string
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
	var projects []string
	for _, p := range m.activeProjects {
		projects = append(projects, p.Name)
	}
	projectsStr = strings.Join(projects, " · ")

	userStyled := ui.InfoPanelUserStyle.Render(userName)
	projectsStyled := ui.InfoPanelProjectStyle.Render(projectsStr)
	line1InnerWidth := m.listLayout.panelContentWidth
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
	return ui.InfoPanelStyle.Render(content)
}

func (m model) renderMetadataPanel(width int) string {
	index := ui.DimTextStyle.Render(
		fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.sections[m.sectionCursor].Issues)),
	)
	var parent string
	if m.issueDetail.Parent != nil {
		parent = ui.RenderIssueType(m.issueDetail.Parent.Type, false) + " " +
			ui.DimTextStyle.Render(m.issueDetail.Parent.Key+" / ")
	}

	issueKey := ui.RenderIssueType(m.issueDetail.Type, false) + " " + ui.DetailHeaderStyle.Render(m.issueDetail.Key)
	summaryMaxWidth := 50
	issueSummary := ui.DetailValueStyle.Render(truncateLongString(m.issueDetail.Summary, summaryMaxWidth))
	var linkedIssue string
	// TODO: make all links available
	if m.issueDetail.IsLinkedToChange {
		linkedIssue = "🔗 " + jira.MonthlyChangeIssue
	}

	detailsHeaderLine1 := index + " " + parent + issueKey + "  " + issueSummary + " " + linkedIssue

	status := ui.RenderStatusBadge(m.issueDetail.Status)
	assignee := ui.DimTextStyle.Render("@" + strings.ToLower(strings.Split(m.issueDetail.Assignee, " ")[0]))

	logged := ""
	if m.issueDetail.Worklogs != nil {
		logged = ui.DimTextStyle.Render("Logged: " + extractLoggedTime(m.issueDetail.Worklogs))
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

	return ui.RenderPanelWithLabel("Metadata", detailsContent.String(), width, m.focusedSection == metadataSection)
}

func (m model) renderStatusBar() string {
	var statusBar strings.Builder
	var style lipgloss.Style

	switch m.statusMessage.msgType {
	case infoStatusBarMsg:
		style = ui.StatusBarInfoStyle
	case errStatusBarMsg:
		style = ui.StatusBarErrorStyle
	}

	if m.loadingCount > 0 {
		if m.statusMessage.content != "" {
			statusBar.WriteString("  " + m.spinner.View() + "  " + m.statusMessage.content)
		} else {
			statusBar.WriteString("  " + m.spinner.View() + "  Loading...")
		}
	} else {
		statusBar.WriteString("  " + m.statusMessage.content)
	}

	return style.Render(statusBar.String())
}

func (m model) buildDescriptionContent(width int) string {
	var content strings.Builder

	if m.issueDetail.Description != nil {
		descText := jira.ExtractText(m.issueDetail.Description, width)
		wrappedDesc := ui.DetailValueStyle.Render(descText)
		content.WriteString(wrappedDesc + "\n\n")
	} else {
		content.WriteString(ui.StatusBarInfoStyle.Render("No description") + "\n\n")
	}
	return content.String()
}

func (m model) renderDescriptionPanel(width int) string {
	viewport := m.descViewport.View()
	return ui.RenderPanelWithLabel("Description", viewport, width, m.focusedSection == descriptionSection)
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
		timestamp += ui.DimTextStyle.Render(" (edited)")
	}

	if isSelected {
		cursor := ui.IconCursor
		comment.WriteString(cursor + ui.SelectedRowStyle.Render(author+timestamp) + "\n")
	} else {
		comment.WriteString(author + timestamp + "\n")
	}

	bodyText := jira.ExtractText(c.Body, width)
	wrappedBody := ui.CommentBodyStyle.MaxWidth(width - ui.PanelOverheadWidth).Render(bodyText)
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
	return ui.RenderPanelWithLabel("Comments", viewport, width, m.focusedSection == commentsSection)
}

func (m model) buildWorklogsContent(width int) string {
	var content strings.Builder
	wlCount := len(m.issueDetail.Worklogs)

	if wlCount > 0 {
		for i, w := range m.issueDetail.Worklogs {
			isSelected := m.worklogsCursor == i
			isLast := i == wlCount-1

			wl := m.renderWorklog(w, width, isSelected, isLast)
			content.WriteString(wl)
		}
	}

	return content.String()
}

func (m model) renderWorklog(w jira.Worklog, width int, isSelected bool, isLast bool) string {
	var wl strings.Builder

	user := m.getUserName(w.Author.AccountID)
	loggedTime := ui.WorklogsAuthorStyle.Render(formatSecondsToString(w.Time))
	author := ui.WorklogsAuthorStyle.Render(user)
	timestamp := ui.WorklogsTimestampStyle.Render(" • " + timeAgo(w.UpdatedAt))
	description := truncateLongString(ui.WorkLogsDescriptionStyle.Render(w.Description), width)

	line1 := loggedTime + " " + author + " " + timestamp
	line2 := description

	if isSelected {
		cursor := ui.IconCursor
		wl.WriteString(cursor + ui.SelectedRowStyle.MaxWidth(width-4).Render(line1) + "\n")
	} else {
		wl.WriteString(ui.NormalRowStyle.MaxWidth(width-4).Render(line1) + "\n")
	}

	wl.WriteString(line2 + "\n")

	if !isLast {
		wl.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
	} else {
		wl.WriteString("\n")
	}

	return wl.String()
}

func (m model) renderWorklogsPanel(width int) string {
	viewport := m.worklogsViewport.View()
	return ui.RenderPanelWithLabel("Worklogs", viewport, width, m.focusedSection == worklogsSection)
}

func (m model) getUserName(accountID string) string {
	for _, u := range m.usersCache {
		if u.ID == accountID {
			return u.Name
		}
	}

	return accountID
}

func (m model) buildSubTasksContent(width int) string {
	var content strings.Builder
	if m.issueDetail != nil {
		sortIssuesByStatus(m.issueDetail.SubTasks)
		subTasksCount := len(m.issueDetail.SubTasks)

		if subTasksCount > 0 {
			for i, c := range m.issueDetail.SubTasks {
				isSelected := m.subTasksCursor == i
				isLast := i == subTasksCount-1

				ch := m.renderSubTask(c, width, isSelected, isLast)
				content.WriteString(ch)
			}
		}
	}

	return content.String()
}

func (m model) renderSubTask(i jira.Issue, width int, isSelected bool, isLast bool) string {
	var content strings.Builder

	issue := ui.RenderIssueType(i.Type, false)
	key := i.Key
	priority := ui.RenderPriority(i.Priority, false)
	status := ui.RenderStatusBadge(i.Status)
	assignee := ui.DimTextStyle.Render("@" + strings.ToLower(strings.Split(i.Assignee, " ")[0]))

	if isSelected {
		cursor := ui.IconCursor
		content.WriteString(cursor + issue + " " + key + " " + priority + " " + status + " " + assignee + "\n")
	} else {
		content.WriteString(issue + " " + key + " " + priority + " " + status + " " + assignee + "\n")
	}

	summary := ui.CommentBodyStyle.Render(truncateLongString(i.Summary, width-5))
	content.WriteString(summary + "\n")

	if !isLast {
		content.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
	} else {
		content.WriteString("\n")
	}

	return content.String()
}

func (m model) renderSubTasksPanel(width int) string {
	viewport := m.subTasksViewport.View()
	return ui.RenderPanelWithLabel("SubTasks", viewport, width, m.focusedSection == subTasksSection)
}

func (m model) renderSimpleBackground() string {
	bg := lipgloss.NewStyle().
		Width(m.windowWidth).
		Height(m.windowHeight).
		Background(lipgloss.Color("0")).
		Render("")

	return bg
}

func (m model) clearStatusAfter(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return clearStatusMsg{}
	}
}
