package config

import "testing"

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
		{name: "missing TEMPO_URL", unset: "TEMPO_URL", wantErr: true},
		{name: "missing TEMPO_TOKEN", unset: "TEMPO_TOKEN", wantErr: true},
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
			if cfg.TempoURL != full["TEMPO_URL"] {
				t.Errorf("TempoURL = %q, want %q", cfg.TempoURL, full["TEMPO_URL"])
			}
			if cfg.TempoToken != full["TEMPO_TOKEN"] {
				t.Errorf("TempoToken = %q, want %q", cfg.TempoToken, full["TEMPO_TOKEN"])
			}
		})
	}
}
