package store

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDirName    = "jira-tui"
	stateFileName = "state.json"
)

// DefaultPath returns $XDG_STATE_HOME/jira-tui/state.json, falling back to
// ~/.local/state/jira-tui/state.json.
func DefaultPath() (string, error) {
	return statePath(os.Getenv("XDG_STATE_HOME"), os.UserHomeDir)
}

func statePath(xdgStateHome string, homeDir func() (string, error)) (string, error) {
	if filepath.IsAbs(xdgStateHome) {
		return filepath.Join(xdgStateHome, appDirName, stateFileName), nil
	}
	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory for the state file: %w", err)
	}
	if home == "" {
		return "", fmt.Errorf("resolving home directory for the state file: %w", os.ErrNotExist)
	}
	return filepath.Join(home, ".local", "state", appDirName, stateFileName), nil
}
