package jira

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newTestClient(handler http.Handler) (*Client, *httptest.Server) {
	srv := httptest.NewServer(handler)
	c, _ := NewClient(srv.URL, "user@example.com", "jira-token", srv.URL, "tempo-token")
	return c, srv
}

func TestSearchIssuesJqlMappingAndPagination(t *testing.T) {
	page1 := `{
		"nextPageToken": "PAGE2",
		"issues": [
			{
				"key": "DEV-1", "id": "1001",
				"fields": {
					"summary": "First issue",
					"status": {"name": "In Progress"},
					"issuetype": {"name": "Task"},
					"project": {"id": "10", "key": "DEV", "name": "Dev"},
					"priority": {"id": "2", "name": "High"},
					"parent": {"id": "999", "key": "DEV-0", "parent_type": "Epic"}
				}
			}
		]
	}`
	page2 := `{
		"nextPageToken": "",
		"issues": [
			{
				"key": "DEV-2", "id": "1002",
				"fields": {
					"summary": "Second issue",
					"status": {"name": "To Do"},
					"issuetype": {"name": "Bug"},
					"project": {"id": "10", "key": "DEV", "name": "Dev"}
				}
			}
		]
	}`

	var calls int
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/search/jql", func(w http.ResponseWriter, r *http.Request) {
		if u, p, ok := r.BasicAuth(); !ok || u == "" || p == "" {
			t.Errorf("expected basic auth to be set, got user=%q ok=%v", u, ok)
		}
		if calls == 0 {
			_, _ = w.Write([]byte(page1))
		} else {
			_, _ = w.Write([]byte(page2))
		}
		calls++
	})

	c, srv := newTestClient(mux)
	defer srv.Close()

	issues, err := c.SearchIssuesJql(context.Background(), "assignee = currentUser()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 2 {
		t.Errorf("expected 2 paginated requests, got %d", calls)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 issues, got %d", len(issues))
	}

	first := issues[0]
	if first.Key != "DEV-1" || first.Summary != "First issue" {
		t.Errorf("unexpected first issue: %+v", first)
	}
	if first.Priority.Name != "High" {
		t.Errorf("first issue priority = %q, want High", first.Priority.Name)
	}
	if first.Parent == nil || first.Parent.Key != "DEV-0" {
		t.Errorf("first issue parent not mapped: %+v", first.Parent)
	}
	if first.Assignee != "Unassigned" {
		t.Errorf("first issue assignee = %q, want Unassigned", first.Assignee)
	}
}

func TestGetIssueDetailMapping(t *testing.T) {
	body := `{
		"key": "DEV-5", "id": "5",
		"fields": {
			"summary": "Detail issue",
			"status": {"name": "In Progress"},
			"issuetype": {"name": "Story"},
			"project": {"id": "10", "key": "DEV", "name": "Dev"},
			"priority": {"id": "1", "name": "Highest"},
			"comment": {
				"comments": [
					{"id": "c1", "author": {"displayName": "Jane"}, "created": "2024-01-01", "updated": "2024-01-02"}
				]
			}
		}
	}`

	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/issue/DEV-5", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})

	c, srv := newTestClient(mux)
	defer srv.Close()

	issue, err := c.GetIssueDetail(context.Background(), "DEV-5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if issue.Key != "DEV-5" || issue.Summary != "Detail issue" {
		t.Errorf("unexpected issue: %+v", issue)
	}
	if issue.Assignee != "Unassigned" {
		t.Errorf("assignee = %q, want Unassigned", issue.Assignee)
	}
	if issue.Priority.Name != "Highest" {
		t.Errorf("priority = %q, want Highest", issue.Priority.Name)
	}
	if len(issue.Comments) != 1 || issue.Comments[0].Author != "Jane" {
		t.Errorf("comments not mapped: %+v", issue.Comments)
	}
}

func TestGetTransitionsMapping(t *testing.T) {
	body := `{
		"transitions": [
			{"id": "11", "name": "Start", "to": {"name": "In Progress"}},
			{"id": "21", "name": "Finish", "to": {"name": "Done"}}
		]
	}`
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/issue/DEV-9/transitions", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})

	c, srv := newTestClient(mux)
	defer srv.Close()

	ts, err := c.GetTransitions(context.Background(), "DEV-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ts) != 2 {
		t.Fatalf("expected 2 transitions, got %d", len(ts))
	}
	// Name is mapped from the target status (to.name), not the transition name.
	if ts[0].Name != "In Progress" || ts[1].Name != "Done" {
		t.Errorf("transition names not mapped from to.name: %+v", ts)
	}
	if ts[0].ID != "11" {
		t.Errorf("transition id = %q, want 11", ts[0].ID)
	}
}

func TestGetAllUsersFiltersNonAtlassian(t *testing.T) {
	body := `[
		{"accountId": "a1", "accountType": "atlassian", "displayName": "Human One", "emailAddress": "h1@x.com"},
		{"accountId": "a2", "accountType": "app", "displayName": "A Bot"},
		{"accountId": "a3", "accountType": "atlassian", "displayName": "Human Two"}
	]`
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/users/search", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})

	c, srv := newTestClient(mux)
	defer srv.Close()

	users, err := c.GetAllUsers(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("expected 2 atlassian users, got %d: %+v", len(users), users)
	}
	for _, u := range users {
		if u.Type != "atlassian" {
			t.Errorf("non-atlassian user leaked through: %+v", u)
		}
	}
}

func TestGetStatuses(t *testing.T) {
	body := `[
		{"statuses": [
			{"id": "1", "name": "To Do", "statusCategory": {"id": 2, "key": "new", "name": "To Do"}},
			{"id": "2", "name": "Done", "statusCategory": {"id": 3, "key": "done", "name": "Done"}}
		]}
	]`
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/project/10/statuses", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(body))
	})

	c, srv := newTestClient(mux)
	defer srv.Close()

	statuses, err := c.GetStatuses(context.Background(), Project{ID: "10", Name: "Dev"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(statuses) != 2 {
		t.Fatalf("expected 2 statuses, got %d", len(statuses))
	}
	if statuses[1].StatusCategory.Key != "done" {
		t.Errorf("status category not mapped: %+v", statuses[1])
	}
}

func TestGetWorkLogsTempoAuth(t *testing.T) {
	body := `{"results": [
		{"tempoWorklogId": 1, "timeSpentSeconds": 3600, "startDate": "2024-01-01", "author": {"accountId": "a1"}}
	]}`
	mux := http.NewServeMux()
	mux.HandleFunc("/4/worklogs/issue/5", func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer tempo-token" {
			t.Errorf("tempo Authorization = %q, want Bearer tempo-token", got)
		}
		_, _ = w.Write([]byte(body))
	})

	c, srv := newTestClient(mux)
	defer srv.Close()

	wls, err := c.GetWorkLogs(context.Background(), "5")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(wls) != 1 || wls[0].Time != 3600 {
		t.Errorf("worklogs not mapped: %+v", wls)
	}
}

func TestDoJiraRequestErrorStatus(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/project/search", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	})

	c, srv := newTestClient(mux)
	defer srv.Close()

	_, err := c.GetProjects(context.Background())
	if err == nil {
		t.Fatalf("expected error for 500 response, got nil")
	}
}
