package store

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStatePath(t *testing.T) {
	t.Parallel()

	home := func(dir string) func() (string, error) {
		return func() (string, error) { return dir, nil }
	}
	broken := func() (string, error) { return "", os.ErrNotExist }

	tests := []struct {
		name    string
		xdg     string
		homeDir func() (string, error)
		want    string
		wantErr bool
		whyItIs string
	}{
		{
			name:    "absolute XDG_STATE_HOME wins",
			xdg:     "/custom/state",
			homeDir: home("/home/u"),
			want:    filepath.Join("/custom/state", appDirName, stateFileName),
		},
		{
			name:    "no XDG falls back to ~/.local/state",
			homeDir: home("/home/u"),
			want:    filepath.Join("/home/u", ".local", "state", appDirName, stateFileName),
		},
		{
			name:    "relative XDG is ignored",
			xdg:     "relative/dir",
			homeDir: home("/home/u"),
			want:    filepath.Join("/home/u", ".local", "state", appDirName, stateFileName),
			whyItIs: "a relative XDG value must not drop state into the working directory, the way debug.log does",
		},
		{
			name:    "a bare dot is ignored too",
			xdg:     ".",
			homeDir: home("/home/u"),
			want:    filepath.Join("/home/u", ".local", "state", appDirName, stateFileName),
		},
		{
			name:    "no home directory",
			homeDir: broken,
			wantErr: true,
		},
		{
			name:    "empty home directory",
			homeDir: home(""),
			wantErr: true,
			whyItIs: "an empty home must not resolve to /.local/state",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := statePath(tt.xdg, tt.homeDir, stateFileName)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("statePath(%q) = %q, want an error: %s", tt.xdg, got, tt.whyItIs)
				}
				return
			}
			if err != nil {
				t.Fatalf("statePath(%q) unexpected error: %v", tt.xdg, err)
			}
			if got != tt.want {
				t.Errorf("statePath(%q) = %q, want %q. %s", tt.xdg, got, tt.want, tt.whyItIs)
			}
		})
	}

	t.Run("home lookup errors are wrapped", func(t *testing.T) {
		t.Parallel()
		_, err := statePath("", broken, stateFileName)
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("error = %v, want it to wrap the home lookup failure", err)
		}
	})
}

func TestDefaultLogPathUsesStateDir(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	got, err := DefaultLogPath()
	if err != nil {
		t.Fatalf("DefaultLogPath() unexpected error: %v", err)
	}

	want := filepath.Join(dir, appDirName, logFileName)
	if got != want {
		t.Errorf("DefaultLogPath() = %q, want %q", got, want)
	}
}
