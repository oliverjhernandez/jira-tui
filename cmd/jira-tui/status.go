package main

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

// messageType controls how a status-bar message is styled.
type messageType int

const (
	infoStatusBarMsg messageType = iota
	errStatusBarMsg
	successStatusBarMsg
)

// statusMessage is the single piece of transient text shown in the status bar.
type statusMessage struct {
	content string
	msgType messageType
}

// setError records a failure: it logs the full error detail (for debug.log)
// and shows a concise, humanized message in the status bar. op describes the
// operation that failed, e.g. "loading issues".
func (m *model) setError(op string, err error) {
	slog.Error(op, "err", err)
	m.statusMessage = statusMessage{
		content: humanizeError(err),
		msgType: errStatusBarMsg,
	}
}

// setErrorMsg shows a user-facing error in the status bar for validation
// failures that already carry a friendly message (no underlying error value).
func (m *model) setErrorMsg(msg string) {
	slog.Warn(msg)
	m.statusMessage = statusMessage{content: msg, msgType: errStatusBarMsg}
}

// setInfo shows a neutral, informational message in the status bar.
func (m *model) setInfo(msg string) {
	m.statusMessage = statusMessage{content: msg, msgType: infoStatusBarMsg}
}

// setSuccess shows a success message in the status bar.
func (m *model) setSuccess(msg string) {
	m.statusMessage = statusMessage{content: msg, msgType: successStatusBarMsg}
}

// humanizeError maps an error to a concise, user-facing message suitable for
// the status bar. The full detail always goes to the log; this never returns a
// raw response body.
func humanizeError(err error) string {
	if err == nil {
		return ""
	}

	var apiErr *jira.APIError
	if errors.As(err, &apiErr) {
		return httpStatusMessage(apiErr.StatusCode)
	}

	// Network-level failures (DNS, refused connections, timeouts).
	var urlErr *url.Error
	var netErr net.Error
	if errors.As(err, &urlErr) || errors.As(err, &netErr) {
		return "cannot reach Jira"
	}

	return err.Error()
}

// httpStatusMessage returns a friendly, one-line description of an HTTP status.
func httpStatusMessage(code int) string {
	switch code {
	case http.StatusUnauthorized:
		return "unauthorized (401) — check your credentials"
	case http.StatusForbidden:
		return "forbidden (403) — insufficient permissions"
	case http.StatusNotFound:
		return "not found (404)"
	case http.StatusTooManyRequests:
		return "rate limited (429) — try again shortly"
	}

	switch {
	case code >= 500:
		return fmt.Sprintf("Jira server error (%d)", code)
	case code >= 400:
		return fmt.Sprintf("request rejected (%d)", code)
	default:
		return fmt.Sprintf("unexpected response (%d)", code)
	}
}
