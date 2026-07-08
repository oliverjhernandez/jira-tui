package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/oliverjhernandez/jira-tui/internal/jira"
)

func TestIsBlockedTransition(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"🔴 BLOQUEADO", true},
		{"Bloqueado", true},
		{"Bloquear", true},
		{"Done", false},
		{"In Progress", false},
		{"Cancelar", false},
	}
	for _, tt := range tests {
		if got := isBlockedTransition(jira.Transition{Name: tt.name}); got != tt.want {
			t.Errorf("isBlockedTransition(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestNewBlockReasonFormData(t *testing.T) {
	fd := NewBlockReasonFormData()
	if fd == nil || fd.Form == nil {
		t.Fatal("NewBlockReasonFormData returned nil form")
	}
	if fd.Reason != "" {
		t.Errorf("default Reason = %q, want empty", fd.Reason)
	}
}

func TestPostBlockedTransitionCmdPayload(t *testing.T) {
	var captured map[string]any

	mux := http.NewServeMux()
	mux.HandleFunc("/rest/api/3/issue/TSIPC-61/transitions", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &captured); err != nil {
			t.Errorf("failed to decode request body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	})

	srv := httptest.NewServer(mux)
	defer srv.Close()

	client, _ := jira.NewClient(srv.URL, "user@example.com", "token", srv.URL, "tempo")
	m := model{client: client}

	msg := m.postBlockedTransitionCmd("TSIPC-61", "3", "server is down")()

	if _, ok := msg.(transitionPostedMsg); !ok {
		t.Fatalf("expected transitionPostedMsg, got %T (%+v)", msg, msg)
	}

	// transition id
	transition, ok := captured["transition"].(map[string]any)
	if !ok || transition["id"] != "3" {
		t.Errorf("transition id not set correctly: %+v", captured["transition"])
	}

	fields, ok := captured["fields"].(map[string]any)
	if !ok {
		t.Fatalf("fields missing or wrong type: %+v", captured["fields"])
	}

	// Flagged: array of {value: Impediment}
	flagged, ok := fields[flaggedFieldID].([]any)
	if !ok || len(flagged) != 1 {
		t.Fatalf("flagged field wrong shape: %+v", fields[flaggedFieldID])
	}
	flag0, ok := flagged[0].(map[string]any)
	if !ok || flag0["value"] != flaggedFieldValue {
		t.Errorf("flagged value = %+v, want value=%q", flagged[0], flaggedFieldValue)
	}

	// Blocker reason: plain string
	if fields[blockReasonFieldID] != "server is down" {
		t.Errorf("block reason = %v, want %q", fields[blockReasonFieldID], "server is down")
	}
}
