package main

import (
	"strings"
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestBuildSearchJQL(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		wantContains  []string
		wantNotHasKey bool
	}{
		{
			name:          "plain word searches summary and description",
			query:         "login",
			wantContains:  []string{`summary ~ "login*"`, `description ~ "login*"`, "ORDER BY updated DESC"},
			wantNotHasKey: true,
		},
		{
			name:         "key-looking query also matches key",
			query:        "DEV-123",
			wantContains: []string{`key = "DEV-123"`, `summary ~ "DEV-123*"`, `description ~ "DEV-123*"`},
		},
		{
			name:         "embedded quote is escaped",
			query:        `a"b`,
			wantContains: []string{`a\"b*`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildSearchJQL(tt.query)
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("buildSearchJQL(%q) = %q, missing %q", tt.query, got, want)
				}
			}
			if tt.wantNotHasKey && strings.Contains(got, "key =") {
				t.Errorf("buildSearchJQL(%q) = %q, should not contain a key clause", tt.query, got)
			}
		})
	}
}

func TestRankSearchResults(t *testing.T) {
	issues := []jira.Issue{
		{Key: "DEV-9", Summary: "unrelated"},                // description match (fallback)
		{Key: "DEV-1", Summary: "fix the login page"},       // summary match
		{Key: "LOGIN-2", Summary: "unrelated"},              // key match
		{Key: "DEV-3", Summary: "another LOGIN in summary"}, // summary match
	}

	got := rankSearchResults(issues, "login")

	if len(got) != 4 {
		t.Fatalf("expected 4 results, got %d", len(got))
	}

	// Order must be: key matches, then summary matches, then description matches.
	wantCats := []searchCategory{catKey, catSummary, catSummary, catDescription}
	for i, want := range wantCats {
		if got[i].category != want {
			t.Errorf("result %d category = %d, want %d (key %s)", i, got[i].category, want, got[i].issue.Key)
		}
	}
	if got[0].issue.Key != "LOGIN-2" {
		t.Errorf("first result should be the key match LOGIN-2, got %s", got[0].issue.Key)
	}
}

func TestPushRecentSearch(t *testing.T) {
	var recents []string
	recents = pushRecentSearch(recents, "a")
	recents = pushRecentSearch(recents, "b")
	recents = pushRecentSearch(recents, "a") // moves "a" to front, dedupes

	if len(recents) != 2 {
		t.Fatalf("expected 2 recents, got %v", recents)
	}
	if recents[0] != "a" || recents[1] != "b" {
		t.Errorf("expected [a b], got %v", recents)
	}

	// blank queries are ignored
	if got := pushRecentSearch(recents, "   "); len(got) != 2 {
		t.Errorf("blank query should be ignored, got %v", got)
	}
}
