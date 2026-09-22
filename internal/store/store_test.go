package store

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func tempStatePath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "jira-tui", "state.json")
}

func TestOpenMissingFileStartsEmpty(t *testing.T) {
	t.Parallel()

	path := tempStatePath(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open on a missing file should not error, got %v", err)
	}
	if got := s.Tags("DEV-1"); got != nil {
		t.Errorf("a fresh store should have no tags, got %v", got)
	}
	if _, err := os.Stat(filepath.Dir(path)); !os.IsNotExist(err) {
		t.Error("Open should not create the state directory; only a save should")
	}
}

func TestRoundTrip(t *testing.T) {
	t.Parallel()

	path := tempStatePath(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SetTags(t.Context(), "DEV-1", []string{"Urgent", "review"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}
	if err := s.SetTags(t.Context(), "OPS-9", []string{"blocked"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("reopening: %v", err)
	}
	if got, want := reopened.Tags("DEV-1"), []string{"review", "urgent"}; !reflect.DeepEqual(got, want) {
		t.Errorf("DEV-1 tags = %v, want %v", got, want)
	}
	if got, want := reopened.AllTags(), []string{"blocked", "review", "urgent"}; !reflect.DeepEqual(got, want) {
		t.Errorf("AllTags = %v, want %v", got, want)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("state file mode = %v, want 0600: it records what the user privately thinks about work issues", got)
	}
}

func TestIssueKeysAreCaseInsensitive(t *testing.T) {
	t.Parallel()

	s, err := Open(tempStatePath(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SetTags(t.Context(), "dev-1", []string{"urgent"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}
	if got := s.Tags("DEV-1"); len(got) != 1 {
		t.Errorf("Tags(DEV-1) = %v, want the tags written as dev-1", got)
	}
}

func TestSetTagsEmptyDropsTheIssue(t *testing.T) {
	t.Parallel()

	path := tempStatePath(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SetTags(t.Context(), "DEV-1", []string{"urgent"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}
	if err := s.SetTags(t.Context(), "DEV-1", nil); err != nil {
		t.Fatalf("clearing: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading state: %v", err)
	}
	if strings.Contains(string(raw), "DEV-1") {
		t.Errorf("clearing the last tag should drop the issue entry, got:\n%s", raw)
	}
}

func TestTagsReturnsACopy(t *testing.T) {
	t.Parallel()

	s, err := Open(tempStatePath(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SetTags(t.Context(), "DEV-1", []string{"urgent"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}

	got := s.Tags("DEV-1")
	got[0] = "mutated"
	if again := s.Tags("DEV-1"); again[0] != "urgent" {
		t.Errorf("Tags must return a copy; the store now reads %v", again)
	}
}

func TestOpenCorruptQuarantinesAndKeepsWorking(t *testing.T) {
	t.Parallel()

	path := tempStatePath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte("{not json"), 0o600); err != nil {
		t.Fatalf("seeding a corrupt file: %v", err)
	}

	s, err := Open(path)
	if !errors.Is(err, ErrCorrupt) {
		t.Fatalf("Open error = %v, want ErrCorrupt", err)
	}
	matches, _ := filepath.Glob(path + ".corrupt-*")
	if len(matches) != 1 {
		t.Fatalf("the unparseable file should be moved aside, found %d sidecars", len(matches))
	}
	if err := s.SetTags(t.Context(), "DEV-1", []string{"urgent"}); err != nil {
		t.Fatalf("a quarantined store should still be writable, got %v", err)
	}
}

func TestOpenNewerVersionRefusesToClobber(t *testing.T) {
	t.Parallel()

	path := tempStatePath(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	original := []byte(`{"version":99,"tags":{"DEV-1":["from-the-future"]}}`)
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatalf("seeding: %v", err)
	}

	s, err := Open(path)
	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("Open error = %v, want ErrUnsupportedVersion", err)
	}
	if err := s.SetTags(t.Context(), "DEV-2", []string{"urgent"}); !errors.Is(err, ErrUnsupportedVersion) {
		t.Fatalf("SetTags error = %v, want ErrUnsupportedVersion", err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading back: %v", err)
	}
	if string(after) != string(original) {
		t.Errorf("a newer schema file must be left untouched, got:\n%s", after)
	}
}

func TestNilStoreIsUsable(t *testing.T) {
	t.Parallel()

	var s *Store
	if got := s.Tags("DEV-1"); got != nil {
		t.Errorf("Tags on a nil store = %v, want nil", got)
	}
	if got := s.AllTags(); got != nil {
		t.Errorf("AllTags on a nil store = %v, want nil", got)
	}
	if got := s.Path(); got != "" {
		t.Errorf("Path on a nil store = %q, want empty", got)
	}
	if err := s.SetTags(t.Context(), "DEV-1", []string{"urgent"}); err != nil {
		t.Errorf("SetTags on a nil store should be a silent no-op, got %v", err)
	}
}

func TestSetTagsRejectsBadInputWithoutWriting(t *testing.T) {
	t.Parallel()

	path := tempStatePath(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SetTags(t.Context(), "DEV-1", []string{"bad!"}); !errors.Is(err, ErrInvalidTag) {
		t.Fatalf("SetTags error = %v, want ErrInvalidTag", err)
	}
	if got := s.Tags("DEV-1"); got != nil {
		t.Errorf("a rejected write should leave the issue untagged, got %v", got)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("a rejected write should not create the state file")
	}
}

func TestSaveCancelledContext(t *testing.T) {
	t.Parallel()

	path := tempStatePath(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if err := s.SetTags(ctx, "DEV-1", []string{"urgent"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("SetTags error = %v, want context.Canceled", err)
	}
	if got := s.Tags("DEV-1"); len(got) != 1 {
		t.Errorf("the in-memory change should survive a failed write, got %v", got)
	}
}

func TestConcurrentAccess(t *testing.T) {
	t.Parallel()

	s, err := Open(tempStatePath(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}

	var wg sync.WaitGroup
	for i := range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			key := "DEV-" + string(rune('a'+i))
			if err := s.SetTags(t.Context(), key, []string{"tag" + string(rune('a'+i))}); err != nil {
				t.Errorf("SetTags: %v", err)
			}
			s.Tags(key)
			s.AllTags()
		}()
	}
	wg.Wait()

	if got := len(s.AllTags()); got != 16 {
		t.Errorf("AllTags = %d tags, want 16", got)
	}
}

func TestWriteLeavesNoTempFiles(t *testing.T) {
	t.Parallel()

	path := tempStatePath(t)
	s, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.SetTags(t.Context(), "DEV-1", []string{"urgent"}); err != nil {
		t.Fatalf("SetTags: %v", err)
	}

	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".*tmp"))
	if len(leftovers) != 0 {
		t.Errorf("atomic writes should clean up after themselves, found %v", leftovers)
	}
}
