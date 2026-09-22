// Package store is jira-tui's local, single-user state file: the things the
// app remembers between runs that Jira knows nothing about.
//
// Concurrent jira-tui processes are last-write-wins. That is deliberate for a
// single user's private annotations on one machine: writes are rare and
// human-paced, and locking would buy little against a conflict window of
// microseconds.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"
)

// Version is the schema version of the state file. Bump it only for a breaking
// change, and add a case to migrate.
const Version = 1

type state struct {
	Version   int                 `json:"version"`
	UpdatedAt time.Time           `json:"updated_at"`
	Tags      map[string][]string `json:"tags"`
}

type Store struct {
	path     string
	readOnly bool

	mu sync.Mutex
	st state
}

func newState() state {
	return state{Version: Version, Tags: map[string][]string{}}
}

// Open loads the state file at path, and always returns a usable Store. A
// non-nil error means the store started empty and the caller should tell the
// user; it is never a reason to refuse to start.
func Open(path string) (*Store, error) {
	s := &Store{path: path, st: newState()}

	raw, err := os.ReadFile(path)
	switch {
	case os.IsNotExist(err):
		return s, nil
	case err != nil:
		return s, fmt.Errorf("reading %s: %w", path, err)
	}

	st, err := migrate(raw)
	switch {
	case err == nil:
		s.st = st
		return s, nil
	case isUnsupportedVersion(err):
		s.readOnly = true
		return s, err
	default:
		quarantine(path)
		return s, err
	}
}

func migrate(raw []byte) (state, error) {
	var probe struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return state{}, fmt.Errorf("reading the schema version: %w", ErrCorrupt)
	}
	if probe.Version == 0 {
		return state{}, fmt.Errorf("missing schema version: %w", ErrCorrupt)
	}
	if probe.Version > Version {
		return state{}, fmt.Errorf("schema version %d: %w", probe.Version, ErrUnsupportedVersion)
	}

	var st state
	if err := json.Unmarshal(raw, &st); err != nil {
		return state{}, fmt.Errorf("decoding the state (%v): %w", err, ErrCorrupt)
	}
	if st.Tags == nil {
		st.Tags = map[string][]string{}
	}
	st.Version = Version
	return st, nil
}

// quarantine moves a state file we could not parse aside, so the next write
// cannot silently replace the user's tags with an empty document.
func quarantine(path string) {
	_ = os.Rename(path, fmt.Sprintf("%s.corrupt-%d", path, time.Now().UnixNano()))
}

func (s *Store) Path() string {
	if s == nil {
		return ""
	}
	return s.path
}

// Tags returns issueKey's tags, sorted. The result is a copy.
func (s *Store) Tags(issueKey string) []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.st.Tags[normalizeIssueKey(issueKey)])
}

// AllTags returns every distinct tag in the store, sorted.
func (s *Store) AllTags() []string {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	seen := map[string]struct{}{}
	for _, tags := range s.st.Tags {
		for _, t := range tags {
			seen[t] = struct{}{}
		}
	}
	return slices.Sorted(maps.Keys(seen))
}

// SetTags replaces issueKey's tags and persists the store. An empty list drops
// the issue's entry entirely. The change is applied in memory before the write,
// so a failed write leaves the session coherent and reports the error.
func (s *Store) SetTags(ctx context.Context, issueKey string, tags []string) error {
	if s == nil {
		return nil
	}

	key := normalizeIssueKey(issueKey)
	if key == "" {
		return fmt.Errorf("empty issue key: %w", ErrInvalidTag)
	}
	normalized, err := NormalizeTags(tags)
	if err != nil {
		return err
	}

	s.mu.Lock()
	if len(normalized) == 0 {
		delete(s.st.Tags, key)
	} else {
		s.st.Tags[key] = normalized
	}
	s.mu.Unlock()

	return s.save(ctx)
}

func (s *Store) save(ctx context.Context) error {
	if s.path == "" {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("saving the state file: %w", err)
	}

	s.mu.Lock()
	if s.readOnly {
		s.mu.Unlock()
		return fmt.Errorf("refusing to overwrite %s: %w", s.path, ErrUnsupportedVersion)
	}
	s.st.UpdatedAt = time.Now().UTC()
	data, err := json.MarshalIndent(s.st, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return fmt.Errorf("encoding the state: %w", err)
	}

	dir, name := filepath.Split(s.path)
	if err := writeFileAtomic(dir, name, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("saving the state to %s: %w", s.path, err)
	}
	return nil
}

func isUnsupportedVersion(err error) bool {
	return errors.Is(err, ErrUnsupportedVersion)
}
