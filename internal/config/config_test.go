package config

import (
	"errors"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	full := map[string]string{
		"JIRA_URL":    "https://jira.example.com",
		"JIRA_TOKEN":  "jira-token",
		"JIRA_EMAIL":  "user@example.com",
		"TEMPO_URL":   "https://tempo.example.com",
		"TEMPO_TOKEN": "tempo-token",
	}

	tests := []struct {
		name    string
		unset   string // env var to blank out; empty means none
		wantErr bool
	}{
		{name: "all vars present", unset: "", wantErr: false},
		{name: "missing JIRA_URL", unset: "JIRA_URL", wantErr: true},
		{name: "missing JIRA_TOKEN", unset: "JIRA_TOKEN", wantErr: true},
		{name: "missing JIRA_EMAIL", unset: "JIRA_EMAIL", wantErr: true},
		{name: "missing TEMPO_URL is allowed", unset: "TEMPO_URL", wantErr: false},
		{name: "missing TEMPO_TOKEN is allowed", unset: "TEMPO_TOKEN", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range full {
				if k == tt.unset {
					t.Setenv(k, "")
				} else {
					t.Setenv(k, v)
				}
			}

			cfg, err := LoadConfig()

			if tt.wantErr {
				if err == nil {
					t.Fatalf("LoadConfig() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("LoadConfig() unexpected error: %v", err)
			}
			if cfg.JiraURL != full["JIRA_URL"] {
				t.Errorf("JiraURL = %q, want %q", cfg.JiraURL, full["JIRA_URL"])
			}
			if cfg.JiraToken != full["JIRA_TOKEN"] {
				t.Errorf("JiraToken = %q, want %q", cfg.JiraToken, full["JIRA_TOKEN"])
			}
			if cfg.JIraEmail != full["JIRA_EMAIL"] {
				t.Errorf("JIraEmail = %q, want %q", cfg.JIraEmail, full["JIRA_EMAIL"])
			}
			wantTempo := tt.unset != "TEMPO_URL" && tt.unset != "TEMPO_TOKEN"
			if got := cfg.TempoEnabled(); got != wantTempo {
				t.Errorf("TempoEnabled() = %v, want %v", got, wantTempo)
			}
			if wantTempo {
				if cfg.TempoURL != full["TEMPO_URL"] {
					t.Errorf("TempoURL = %q, want %q", cfg.TempoURL, full["TEMPO_URL"])
				}
				if cfg.TempoToken != full["TEMPO_TOKEN"] {
					t.Errorf("TempoToken = %q, want %q", cfg.TempoToken, full["TEMPO_TOKEN"])
				}
			}
		})
	}
}

func TestMissingEnvErrorMessage(t *testing.T) {
	tests := []struct {
		vars []string
		want string
	}{
		{[]string{"JIRA_URL"}, "missing env var JIRA_URL"},
		{[]string{"JIRA_URL", "JIRA_TOKEN"}, "missing env vars JIRA_URL and JIRA_TOKEN"},
		{[]string{"JIRA_URL", "JIRA_TOKEN", "JIRA_EMAIL"}, "missing env vars JIRA_URL, JIRA_TOKEN and JIRA_EMAIL"},
	}

	for _, tt := range tests {
		err := &MissingEnvError{Vars: tt.vars}
		if got := err.Error(); got != tt.want {
			t.Errorf("Error() = %q, want %q", got, tt.want)
		}
	}
}

func TestLoadConfigReportsAllMissingVars(t *testing.T) {
	t.Setenv("JIRA_URL", "")
	t.Setenv("JIRA_TOKEN", "")
	t.Setenv("JIRA_EMAIL", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected an error when every required var is unset")
	}

	var missing *MissingEnvError
	if !errors.As(err, &missing) {
		t.Fatalf("expected *MissingEnvError, got %T", err)
	}
	if len(missing.Vars) != 3 {
		t.Errorf("expected all 3 missing vars reported, got %v", missing.Vars)
	}
}
