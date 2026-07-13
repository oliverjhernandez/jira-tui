package main

import (
	"errors"
	"fmt"
	"net"
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
