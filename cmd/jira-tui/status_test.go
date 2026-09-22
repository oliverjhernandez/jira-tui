package main

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestHumanizeError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want string
	}{
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "api 401",
			err:  &jira.APIError{StatusCode: 401, Body: `{"errorMessages":["secret"]}`},
			want: "unauthorized (401) — check your credentials",
		},
		{
			name: "api 404",
			err:  &jira.APIError{StatusCode: 404},
			want: "not found (404)",
		},
		{
			name: "api 500",
			err:  &jira.APIError{StatusCode: 500},
			want: "Jira server error (500)",
		},
		{
			name: "api 418 (generic 4xx)",
			err:  &jira.APIError{StatusCode: 418},
			want: "request rejected (418)",
		},
		{
			name: "wrapped api error",
			err:  fmt.Errorf("fetching issues: %w", &jira.APIError{StatusCode: 403}),
			want: "forbidden (403) — insufficient permissions",
		},
		{
			name: "url error",
			err:  &url.Error{Op: "Get", URL: "https://jira", Err: errors.New("no such host")},
			want: "cannot reach Jira",
		},
		{
			name: "net error",
			err:  &net.OpError{Op: "dial", Err: errors.New("connection refused")},
			want: "cannot reach Jira",
		},
		{
			name: "plain error falls through",
			err:  errors.New("something broke"),
			want: "something broke",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := humanizeError(tt.err); got != tt.want {
				t.Errorf("humanizeError() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestAPIErrorIncludesBody(t *testing.T) {
	t.Parallel()

	err := &jira.APIError{Method: "POST", Endpoint: "/rest/api/3/issue", StatusCode: 400, Body: "bad field"}
	got := err.Error()
	for _, want := range []string{"POST", "/rest/api/3/issue", "400", "bad field"} {
		if !strings.Contains(got, want) {
			t.Errorf("APIError.Error() = %q, missing %q", got, want)
		}
	}
}

func TestAuthErrorsAreSticky(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		err        error
		wantSticky bool
	}{
		{"401 sticks", &jira.APIError{StatusCode: http.StatusUnauthorized}, true},
		{"403 sticks", &jira.APIError{StatusCode: http.StatusForbidden}, true},
		{"404 clears", &jira.APIError{StatusCode: http.StatusNotFound}, false},
		{"500 clears", &jira.APIError{StatusCode: http.StatusInternalServerError}, false},
		{"plain error clears", errors.New("boom"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var m model
			m.setError("loading issues", tt.err)
			if m.statusMessage.sticky != tt.wantSticky {
				t.Errorf("sticky = %v, want %v", m.statusMessage.sticky, tt.wantSticky)
			}
		})
	}
}

func TestStickyMessageSurvivesClear(t *testing.T) {
	t.Parallel()

	var m model
	m.setError("loading issues", &jira.APIError{StatusCode: http.StatusUnauthorized})

	updated, _ := m.update(clearStatusMsg{})
	got := updated.(model)
	if got.statusMessage.content == "" {
		t.Error("an auth error must stay on screen after the clear tick")
	}

	got.setInfo("something else")
	if got.statusMessage.sticky {
		t.Error("a later message must not inherit stickiness")
	}
}
