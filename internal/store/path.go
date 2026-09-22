package store

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDirName    = "jira-tui"
	stateFileName = "state.json"
	logFileName   = "debug.log"
)

// DefaultPath returns $XDG_STATE_HOME/jira-tui/state.json, falling back to
// ~/.local/state/jira-tui/state.json.
func DefaultPath() (string, error) {
	return statePath(os.Getenv("XDG_STATE_HOME"), os.UserHomeDir, stateFileName)
}

// DefaultLogPath returns $XDG_STATE_HOME/jira-tui/debug.log, falling back to
// ~/.local/state/jira-tui/debug.log.
func DefaultLogPath() (string, error) {
	return statePath(os.Getenv("XDG_STATE_HOME"), os.UserHomeDir, logFileName)
}

func statePath(xdgStateHome string, homeDir func() (string, error), name string) (string, error) {
	if filepath.IsAbs(xdgStateHome) {
		return filepath.Join(xdgStateHome, appDirName, name), nil
	}
	home, err := homeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory for %s: %w", name, err)
	}
	if home == "" {
		return "", fmt.Errorf("resolving home directory for %s: %w", name, os.ErrNotExist)
	}
	return filepath.Join(home, ".local", "state", appDirName, name), nil
}
