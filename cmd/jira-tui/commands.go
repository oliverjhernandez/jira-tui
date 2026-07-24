package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type detailLayout struct {
	leftColumnWidth  int
	rightColumnWidth int
	metadataHeight   int
	descHeight       int
	commentsHeight   int
	worklogsHeight   int
	issueLinksHeight int
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
	tabID  int
}

type subTasksLoadedMsg struct {
	subTasks []jira.Issue
	tabID    int
}

type myselfLoadedMsg struct {
	me *jira.User
}

type statusesLoadedMsg struct {
	statuses map[string][]jira.Status
	tabID    int
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
	detail *jira.Issue
	tabID  int
}

type transitionsLoadedMsg struct {
	status      string
	issueKey    string
	transitions []jira.Transition
}

type usersLoadedMsg struct {
	users []jira.User
}

type workLogsLoadedMsg struct {
	workLogs []jira.Worklog
	tabID    int
}

type worklogTotalsLoadedMsg struct {
	totals map[string]int // issue ID -> total seconds
	tabID  int
}

type transitionPostedMsg struct{}

type assigneePostedMsg struct{}

type newIssuePostedMsg struct{}

type issueLinkPostedMsg struct{}

type updatedDescriptionMsg struct{}

type updatedSummaryMsg struct{}

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

		return issueDetailLoadedMsg{detail: detail, tabID: m.activeTabID()}
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

func (m model) fetchTransitionsCmd(issueKey, status string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		transitions, err := m.client.GetTransitions(context.Background(), issueKey)
		if err != nil {
			return errMsg{err}
		}

		return transitionsLoadedMsg{
			status:      status,
			issueKey:    issueKey,
			transitions: transitions,
		}
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
		if m.myself != nil && data.AssigneeName == m.myself.Name {
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

		description := jira.MarkdownToADF(data.Description)

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

// postTransitionCmd posts a transition. worklogTime, when non-empty (e.g.
// "1h 30m"), is attached as a native Jira worklog to satisfy transitions whose
// screen has a required Time Spent field.
func (m model) postTransitionCmd(issueKey, transitionID, worklogTime string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		if err := m.client.PostTransition(context.Background(), issueKey, transitionID, nil, "", worklogTime); err != nil {
			return errMsg{err}
		}

		return transitionPostedMsg{}
	}
}

const (
	flaggedFieldID     = "customfield_10021"
	flaggedFieldValue  = "Impediment"
	blockReasonFieldID = "customfield_10485"
)

func (m model) postBlockedTransitionCmd(issueKey, transitionID, reason string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		fields := map[string]any{
			flaggedFieldID: []map[string]string{
				{"value": flaggedFieldValue},
			},
			blockReasonFieldID: reason,
		}

		err := m.client.PostTransition(context.Background(), issueKey, transitionID, fields, "", "")
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

func (m model) updateSummaryCmd(issueKey, summary string) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.UpdateSummary(context.Background(), issueKey, summary)
		if err != nil {
			return errMsg{err}
		}

		return updatedSummaryMsg{}
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

// fetchMyIssuesCmd fetches the active tab's board (its JQL), tagging the result
// with the active tab's id.
func (m model) fetchMyIssuesCmd() tea.Cmd {
	return m.fetchBoardIssuesCmd(m.activeBoardJQL(), m.activeTabID())
}

// fetchBoardIssuesCmd fetches issues for an arbitrary board JQL and tags the
// result with the given tab id, so it can be routed to the right tab.
func (m model) fetchBoardIssuesCmd(jql string, tabID int) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		issues, err := m.client.SearchIssuesJql(context.Background(), jql)
		if err != nil {
			return errMsg{err}
		}

		return issuesLoadedMsg{issues: issues, tabID: tabID}
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

		return subTasksLoadedMsg{subTasks: subTasks, tabID: m.activeTabID()}
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

func (m model) fetchStatusesCmd(projects []jira.Project, tabID int) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		results := make(map[string][]jira.Status)
		for _, p := range projects {
			statuses, err := m.client.GetStatuses(context.Background(), p)
			if err != nil {
				return errMsg{err}
			}

			results[p.ID] = statuses
		}

		return statusesLoadedMsg{statuses: results, tabID: tabID}
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

		return workLogsLoadedMsg{workLogs: wls, tabID: m.activeTabID()}
	}
}

func (m model) fetchAllWorklogsTotalCmd(issues []jira.Issue, tabID int) tea.Cmd {
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

		return worklogTotalsLoadedMsg{totals: totals, tabID: tabID}
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

func (m model) postLinkIssueCmd(fromKey, toKey string, linkType jira.LinkType) tea.Cmd {
	return func() tea.Msg {
		if m.client == nil {
			return errMsg{fmt.Errorf("jira client not initialized")}
		}

		err := m.client.PostIssueLink(context.Background(), fromKey, toKey, linkType)
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

func (m *model) classifyIssues(issues []jira.Issue, statuses map[string][]jira.Status) []Section {
	sections := []Section{
		{Name: "In Progress", CategoryKey: "indeterminate"},
		{Name: "To Do", CategoryKey: "new"},
		{Name: "In Transit", CategoryKey: "transit"},
		{Name: "Done", CategoryKey: "done", Collapsed: true},
	}

	global := make(map[string]string)
	for _, projStatuses := range statuses {
		for _, s := range projStatuses {
			global[strings.ToLower(strings.TrimSpace(s.Name))] = s.StatusCategory.Key
		}
	}

	var unclassified []jira.Issue

	for i := range issues {
		issue := issues[i]
		key := strings.ToLower(strings.TrimSpace(issue.Status))

		categoryKey := ""
		for _, s := range statuses[issue.Project.ID] {
			if strings.ToLower(strings.TrimSpace(s.Name)) == key {
				categoryKey = s.StatusCategory.Key
				break
			}
		}
		if categoryKey == "" {
			categoryKey = global[key]
		}
		if intransitStatuses[issue.Status] {
			categoryKey = "transit"
		}

		placed := false
		for idx := range sections {
			if sections[idx].CategoryKey == categoryKey {
				sections[idx].Issues = append(sections[idx].Issues, issue)
				placed = true
				break
			}
		}
		if !placed {
			unclassified = append(unclassified, issue)
			slog.Warn("unclassified issue", "key", issue.Key, "status", issue.Status, "categoryKey", categoryKey)
		}
	}

	if len(unclassified) > 0 {
		sections = append(sections, Section{Name: "Other", CategoryKey: "other", Issues: unclassified})
	}

	return sections
}

func (m model) calculateDetailLayout() detailLayout {
	panelWidth := m.windowWidth
	leftColumnWidth := int(float64(panelWidth) * 0.8)
	rightColumnWidth := int(float64(panelWidth) * 0.2)

	metadataHeight := 7
	statusBarHeight := 1

	leftFixedHeight := metadataHeight + statusBarHeight + tabBarHeight + (ui.PanelOverheadHeight * 2)
	rightFixedHeight := statusBarHeight + tabBarHeight + (ui.PanelOverheadHeight * 3)
	leftColumnFreeHeight := m.windowHeight - leftFixedHeight
	rightColumnFreeHeight := m.windowHeight - rightFixedHeight

	descHeight := leftColumnFreeHeight / 2
	commentsHeight := leftColumnFreeHeight / 2

	worklogsHeight := rightColumnFreeHeight / 3
	issueLinksHeight := rightColumnFreeHeight / 3
	subTasksHeight := rightColumnFreeHeight / 3

	return detailLayout{
		leftColumnWidth,
		rightColumnWidth,
		metadataHeight,
		descHeight,
		commentsHeight,
		worklogsHeight,
		issueLinksHeight,
		subTasksHeight,
	}
}

func (m model) calculateListLayout() listLayout {
	infoHeight := 5
	statusBarHeight := 1
	listHeight := m.windowHeight - infoHeight - statusBarHeight - tabBarHeight - ui.PanelOverheadHeight - listHeaderHeight
	panelWidth := m.windowWidth - ui.PanelOverheadWidth

	return listLayout{
		panelWidth,
		infoHeight,
		listHeight,
		statusBarHeight,
	}
}

func (m model) buildListContent() string {
	if m.currentGrouping() == groupEpic {
		return m.buildEpicListContent()
	}

	var listContent strings.Builder

	sortSectionsIssues(m.sections)
	sectionsToRender := m.sections
	if m.filteredSections != nil {
		sectionsToRender = m.filteredSections
	}

	for si, s := range sectionsToRender {
		sectionHeader := ui.SectionTitleStyle.Render(fmt.Sprintf("%s (%d)", s.Name, len(s.Issues)))
		fmt.Fprintf(&listContent, "%s\n", sectionHeader)
		for ii, issue := range s.Issues {
			selected := m.sectionCursor == si && m.cursor == ii
			dimmed := closureStatuses[issue.Status]
			listContent.WriteString(m.renderIssueRow(issue, selected, dimmed) + "\n")
		}
		listContent.WriteString("\n\n")
	}

	return listContent.String()
}

func (m model) renderInfoPanel() string {
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

func (m model) renderMetadataPanel(width int, height int) string {
	if m.activeIssue == nil {
		return ui.RenderPanelWithLabel("Metadata", "", width, height, m.focusedSection == metadataSection)
	}

	var index string
	if m.sectionCursor >= 0 && m.sectionCursor < len(m.sections) {
		index = ui.DimTextStyle.Render(
			fmt.Sprintf("[%d/%d]", m.cursor+1, len(m.sections[m.sectionCursor].Issues)),
		)
	}

	var parent string
	if m.activeIssue.Parent != nil {
		parent = ui.RenderIssueType(m.activeIssue.Parent.Type, false) + " " +
			ui.DimTextStyle.Render(m.activeIssue.Parent.Key+" / ")
	}

	issueKey := ui.RenderIssueType(m.activeIssue.Type, false) + " " + ui.DetailHeaderStyle.Render(m.activeIssue.Key)
	summaryMaxWidth := 50
	issueSummary := ui.DetailValueStyle.Render(ui.TruncateLongString(m.activeIssue.Summary, summaryMaxWidth))
	detailsHeaderLine1 := index + " " + parent + issueKey + "  " + issueSummary

	status := ui.RenderStatusBadge(m.activeIssue.Status)
	assignee := ui.DimTextStyle.Render("@" + strings.ToLower(strings.Split(m.activeIssue.Assignee, " ")[0]))
	logged := ""
	if m.activeIssue.Worklogs != nil {
		logged = ui.DimTextStyle.Render("Logged: " + extractLoggedTime(m.activeIssue.Worklogs))
	}
	detailsHeaderLine2 := status + "  " + assignee + "  " + logged
	leftHeader := detailsHeaderLine1 + "\n" + detailsHeaderLine2

	colwidth := 30
	col1 := ui.RenderFieldStyled("Priority", ui.RenderPriority(m.activeIssue.Priority.Name, true), colwidth)
	// TODO: map reporter to name
	col2 := ui.RenderFieldStyled("Reporter", m.activeIssue.Reporter.DisplayName, colwidth)
	col3 := ui.RenderFieldStyled("Type", ui.RenderIssueType(m.activeIssue.Type, true), colwidth)
	metadataRow1 := lipgloss.JoinHorizontal(lipgloss.Top, col1, col2, col3)

	col4 := ui.RenderFieldStyled("Created", timeAgo(m.activeIssue.Created), colwidth)
	col5 := ui.RenderFieldStyled("Updated", timeAgo(m.activeIssue.Updated), colwidth)
	metadataRow2 := lipgloss.JoinHorizontal(lipgloss.Top, col4, col5)

	var detailsContent strings.Builder
	detailsContent.WriteString(leftHeader + "\n")
	detailsContent.WriteString(metadataRow1 + "\n" + metadataRow2)

	return ui.RenderPanelWithLabel("Metadata", detailsContent.String(), width, height, m.focusedSection == metadataSection)
}

func (m model) renderStatusBar() string {
	if m.filtering {
		return ui.StatusBarInfoStyle.Render("  Filter: " + m.textInput.Value())
	}

	var style lipgloss.Style
	switch m.statusMessage.msgType {
	case errStatusBarMsg:
		style = ui.StatusBarErrorStyle
	case successStatusBarMsg:
		style = ui.StatusBarSuccessStyle
	default:
		style = ui.StatusBarInfoStyle
	}

	// Prefix an icon so severity reads at a glance.
	content := m.statusMessage.content
	if content != "" {
		switch m.statusMessage.msgType {
		case errStatusBarMsg:
			content = "✗ " + content
		case successStatusBarMsg:
			content = "✓ " + content
		}
	}

	var text string
	switch {
	case m.loadingCount > 0 && content != "":
		text = "  " + m.spinner.View() + "  " + content
	case m.loadingCount > 0:
		text = "  " + m.spinner.View() + "  Loading..."
	default:
		text = "  " + content
	}

	// Truncate to the terminal width so long messages never overflow the bar.
	if m.windowWidth > 0 {
		text = ansi.Truncate(text, m.windowWidth, "…")
	}

	return style.Render(text)
}

func (m model) buildDescriptionContent(width int) string {
	var content strings.Builder

	if m.activeIssue.Description != nil {
		descText := jira.ExtractText(m.activeIssue.Description, width)
		content.WriteString(descText + "\n\n")
	} else {
		content.WriteString(ui.StatusBarInfoStyle.Render("No description") + "\n\n")
	}
	return content.String()
}

func (m model) renderDescriptionPanel(width int, height int) string {
	viewport := m.descViewport.View()
	return ui.RenderPanelWithLabel("Description", viewport, width, height, m.focusedSection == descriptionSection)
}

func (m model) buildCommentsContent(width int) string {
	var content strings.Builder
	comments := m.activeIssue.Comments
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
	wrappedBody := ui.CommentBodyStyle.Render(bodyText)
	comment.WriteString(wrappedBody + "\n")

	if !isLast {
		comment.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
	} else {
		comment.WriteString("\n")
	}

	return comment.String()
}

func (m model) renderCommentsPanel(width int, height int) string {
	viewport := m.commentsViewport.View()
	return ui.RenderPanelWithLabel("Comments", viewport, width, height, m.focusedSection == commentsSection)
}

func (m model) buildWorklogsContent(width int) string {
	var content strings.Builder
	wlCount := len(m.activeIssue.Worklogs)

	if wlCount > 0 {
		for i, w := range m.activeIssue.Worklogs {
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
	description := ui.TruncateLongString(ui.WorkLogsDescriptionStyle.Render(w.Description), width)

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

func (m model) renderWorklogsPanel(width int, height int) string {
	viewport := m.worklogsViewport.View()
	return ui.RenderPanelWithLabel("Worklogs", viewport, width, height, m.focusedSection == worklogsSection)
}

func (m model) buildIssueLinksContent(width int) string {
	var content strings.Builder
	ilCount := len(m.activeIssue.IssueLinks)

	if ilCount > 0 {
		for i, l := range m.activeIssue.IssueLinks {
			isSelected := m.IssueLinksCursor == i
			isLast := i == ilCount-1

			il := m.renderIssueLink(l, width, isSelected, isLast)
			content.WriteString(il)
		}
	}

	return content.String()
}

func (m model) renderIssueLinksPanel(width int, height int) string {
	viewport := m.issueLinksViewport.View()
	return ui.RenderPanelWithLabel("Issue Links", viewport, width, height, m.focusedSection == issueLinksSection)
}

func (m model) getUserName(accountID string) string {
	for _, u := range m.usersCache {
		if u.ID == accountID {
			return u.Name
		}
	}

	return accountID
}

func (m model) renderIssueLink(l jira.IssueLink, width int, isSelected bool, isLast bool) string {
	var content strings.Builder

	if l.InwardIssue != nil {
		linkType := ui.DimTextStyle.Render(l.Type.Inward)
		key := ui.KeyFieldStyle.Render(l.InwardIssue.Key)
		content.WriteString(linkType + " " + key + "\n")
	} else if l.OutwardIssue != nil {
		linkType := ui.DimTextStyle.Render(l.Type.Outward)
		key := ui.KeyFieldStyle.Render(l.OutwardIssue.Key)
		content.WriteString(linkType + " " + key + "\n")
	}

	// if isSelected {
	// 	cursor := ui.IconCursor
	// 	content.WriteString(cursor + issue + " " + key + " " + priority + " " + status + " " + assignee + "\n")
	// } else {
	// 	content.WriteString(issue + " " + key + " " + priority + " " + status + " " + assignee + "\n")
	// }

	if !isLast {
		content.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
	} else {
		content.WriteString("\n")
	}

	return content.String()
}

func (m model) buildSubTasksContent(width int) string {
	var content strings.Builder
	if m.activeIssue != nil {
		sortIssuesByStatus(m.activeIssue.SubTasks)
		subTasksCount := len(m.activeIssue.SubTasks)

		if subTasksCount > 0 {
			for i, c := range m.activeIssue.SubTasks {
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
	key := ui.KeyFieldStyle.Render(i.Key)
	priority := ui.RenderPriority(i.Priority.Name, false)
	status := ui.RenderStatusBadge(i.Status)
	assignee := ui.DimTextStyle.Render("@" + strings.ToLower(strings.Split(i.Assignee, " ")[0]))

	if isSelected {
		cursor := ui.IconCursor
		content.WriteString(cursor + issue + " " + key + " " + priority + " " + status + " " + assignee + "\n")
	} else {
		content.WriteString(issue + " " + key + " " + priority + " " + status + " " + assignee + "\n")
	}

	summary := ui.CommentBodyStyle.Render(ui.TruncateLongString(i.Summary, width-5))
	content.WriteString(summary + "\n")

	if !isLast {
		content.WriteString(ui.SeparatorStyle.Render("  ────") + "\n\n")
	} else {
		content.WriteString("\n")
	}

	return content.String()
}

func (m model) renderSubTasksPanel(width int, height int) string {
	viewport := m.subTasksViewport.View()
	return ui.RenderPanelWithLabel("SubTasks", viewport, width, height, m.focusedSection == subTasksSection)
}

func (m model) renderSimpleBackground() string {
	bg := lipgloss.NewStyle().
		Width(m.windowWidth).
		Height(m.windowHeight).
		Background(lipgloss.Color("0")).
		Render("")

	return bg
}

// renderBackground renders the base (full-screen) view a modal is drawn over.
func (m model) renderBackground() string {
	switch m.baseView {
	case detailView:
		return m.renderDetailView()
	default:
		return m.renderListView()
	}
}

// renderModal composites a centered, labeled panel over the current base view.
// wScale and hScale are fractions of the window width/height.
func (m model) renderModal(label, content string, wScale, hScale float64) string {
	modalWidth := ui.GetModalWidth(m.windowWidth, wScale)
	modalHeight := ui.GetModalHeight(m.windowHeight, hScale)

	styledModal := ui.RenderPanelWithLabel(label, content, modalWidth, modalHeight, true)

	x := (m.windowWidth - modalWidth) / 2
	y := (m.windowHeight - modalHeight) / 2

	bg := lipgloss.NewLayer(m.renderBackground())
	fg := lipgloss.NewLayer(styledModal).X(x).Y(y).Z(1)

	return lipgloss.NewCompositor(bg, fg).Render()
}

func (m model) clearStatusAfter(d time.Duration) tea.Cmd {
	return func() tea.Msg {
		time.Sleep(d)
		return clearStatusMsg{}
	}
}
