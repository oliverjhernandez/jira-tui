package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestRenderMetadataPanel_NilIssueDetail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderMetadataPanel panicked with nil activeIssue: %v", r)
		}
	}()

	m := model{}
	m.renderMetadataPanel(80, 8)
}

func TestRenderMetadataPanel_NilParent(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderMetadataPanel panicked with nil Parent: %v", r)
		}
	}()

	m := model{
		activeIssue: &jira.Issue{
			Key:     "DEV-123",
			Summary: "Test issue",
			Status:  "In Progress",
			Type:    "Task",
		},
	}
	m.renderMetadataPanel(80, 8)
}

func TestBuildDescriptionContent_NilDescription(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("buildDescriptionContent panicked with nil Description: %v", r)
		}
	}()

	m := model{
		activeIssue: &jira.Issue{},
	}
	m.buildDescriptionContent(80)
}

func TestBuildCommentsContent_EmptyComments(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("buildCommentsContent panicked with empty comments: %v", r)
		}
	}()

	m := model{
		activeIssue: &jira.Issue{
			Comments: nil,
		},
	}
	m.buildCommentsContent(80)
}

func TestBuildWorklogsContent_NilWorklogs(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("buildWorklogsContent panicked with nil worklogs: %v", r)
		}
	}()

	m := model{
		activeIssue: &jira.Issue{},
	}
	m.buildWorklogsContent(80)
}

func TestBuildIssueLinksContent_NilOutwardIssue(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderIssueLink panicked with nil OutwardIssue: %v", r)
		}
	}()

	m := model{
		activeIssue: &jira.Issue{
			IssueLinks: []jira.IssueLink{
				{
					Type: jira.Link{Outward: "blocks"},
					// Both InwardIssue and OutwardIssue nil
				},
			},
		},
	}
	m.buildIssueLinksContent(80)
}

func TestBuildSubTasksContent_NilIssueDetail(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("buildSubTasksContent panicked with nil activeIssue: %v", r)
		}
	}()

	m := model{}
	m.buildSubTasksContent(80)
}

func TestRenderInfoPanel_NilMyself(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderInfoPanel panicked with nil myself: %v", r)
		}
	}()

	m := model{}
	m.renderInfoPanel()
}
