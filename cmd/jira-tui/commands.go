package main

import (
	"context"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

const (
	listView viewMode = iota
	detailView
	transitionView
	editDescriptionView
	editPriorityView
)

// bubbletea messages from commands
type dataLoadedMsg struct {
	issues     []jira.Issue
	priorities []jira.Priority
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

type editedPriorityMsg struct {
	success bool
}

type errMsg struct {
	err error
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

func (m model) fetchData() tea.Msg {
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

	priorities, err := client.GetPriorities(context.Background())
	if err != nil {
		return errMsg{err}
	}

	return dataLoadedMsg{issues, priorities}
}
