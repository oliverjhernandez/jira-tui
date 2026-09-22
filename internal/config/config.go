// Package config
package config

import (
	"fmt"
	"os"
)

type Config struct {
	JiraURL    string
	JiraToken  string
	TempoURL   string
	TempoToken string
	JIraEmail  string
}

// TempoEnabled reports whether Tempo worklog credentials were supplied. Tempo is
// a paid add-on, so the app runs without it and hides time-tracking features.
func (c *Config) TempoEnabled() bool {
	return c.TempoURL != "" && c.TempoToken != ""
}

func LoadConfig() (*Config, error) {
	cfg := &Config{
		JiraURL:    os.Getenv("JIRA_URL"),
		JiraToken:  os.Getenv("JIRA_TOKEN"),
		JIraEmail:  os.Getenv("JIRA_EMAIL"),
		TempoURL:   os.Getenv("TEMPO_URL"),
		TempoToken: os.Getenv("TEMPO_TOKEN"),
	}

	var missing []string
	if cfg.JiraURL == "" {
		missing = append(missing, "JIRA_URL")
	}
	if cfg.JiraToken == "" {
		missing = append(missing, "JIRA_TOKEN")
	}
	if cfg.JIraEmail == "" {
		missing = append(missing, "JIRA_EMAIL")
	}

	if len(missing) > 0 {
		return nil, &MissingEnvError{Vars: missing}
	}

	return cfg, nil
}

// MissingEnvError reports the required environment variables that were unset.
type MissingEnvError struct {
	Vars []string
}

func (e *MissingEnvError) Error() string {
	if len(e.Vars) == 1 {
		return fmt.Sprintf("missing env var %s", e.Vars[0])
	}
	return fmt.Sprintf("missing env vars %s", joinAnd(e.Vars))
}

func joinAnd(vars []string) string {
	switch len(vars) {
	case 0:
		return ""
	case 1:
		return vars[0]
	case 2:
		return vars[0] + " and " + vars[1]
	}
	out := ""
	for i, v := range vars[:len(vars)-1] {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out + " and " + vars[len(vars)-1]
}
