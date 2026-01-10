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

func LoadConfig() (*Config, error) {
	cfg := &Config{
		JiraURL:    os.Getenv("JIRA_URL"),
		JiraToken:  os.Getenv("JIRA_TOKEN"),
		JIraEmail:  os.Getenv("JIRA_EMAIL"),
		TempoURL:   os.Getenv("TEMPO_URL"),
		TempoToken: os.Getenv("TEMPO_TOKEN"),
	}

	if cfg.JiraURL == "" {
		return nil, fmt.Errorf("missing env var JIRA_URL")
	}

	if cfg.JiraToken == "" {
		return nil, fmt.Errorf("missing env var JIRA_TOKEN")
	}

	if cfg.JIraEmail == "" {
		return nil, fmt.Errorf("missing env var JIRA_EMAIL")
	}

	if cfg.TempoURL == "" {
		return nil, fmt.Errorf("missing env var TEMPO_URL")
	}

	if cfg.TempoToken == "" {
		return nil, fmt.Errorf("missing env var TEMPO_TOKEN")
	}

	return cfg, nil
}
