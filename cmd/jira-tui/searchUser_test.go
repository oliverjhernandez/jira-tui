package main

import (
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestRenderSearchUserView_NilActiveIssue(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("renderSearchUserView panicked with nil activeIssue: %v", r)
		}
	}()

	m := model{
		userSelectionMode: insertMention,
		previousMode:      listView,
		searchUserData:    NewSearchUserFormData([]jira.User{}),
	}
	m.renderSearchUserView()
}
