package store

import (
	"errors"
	"fmt"
	"slices"
	"strings"
)

const (
	MaxTagLen       = 32
	MaxTagsPerIssue = 10
)

var (
	ErrEmptyTag           = errors.New("tag is empty")
	ErrInvalidTag         = errors.New("tag has invalid characters")
	ErrTagTooLong         = errors.New("tag is too long")
	ErrTooManyTags        = errors.New("too many tags on this issue")
	ErrCorrupt            = errors.New("state file is corrupt")
	ErrUnsupportedVersion = errors.New("state file was written by a newer version")
)

// NormalizeTag canonicalizes a user-entered tag: "  Needs Review " becomes
// "needs-review". Tags are lowercase ASCII alphanumerics plus '-', '_', '.'
// and '/', start with a letter or digit, and are at most MaxTagLen long.
func NormalizeTag(raw string) (string, error) {
	t := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(raw), "#")))
	if t == "" {
		return "", fmt.Errorf("%q: %w", raw, ErrEmptyTag)
	}
	t = strings.Join(strings.Fields(t), "-")

	for _, r := range t {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
		case r == '-', r == '_', r == '.', r == '/':
		default:
			return "", fmt.Errorf("%q: %w", raw, ErrInvalidTag)
		}
	}
	if c := t[0]; (c < 'a' || c > 'z') && (c < '0' || c > '9') {
		return "", fmt.Errorf("%q must start with a letter or digit: %w", raw, ErrInvalidTag)
	}
	if len(t) > MaxTagLen {
		return "", fmt.Errorf("%q is %d characters, max %d: %w", raw, len(t), MaxTagLen, ErrTagTooLong)
	}
	return t, nil
}

// NormalizeTags normalizes every tag, dropping duplicates and sorting the
// result. It reports the first rejected tag rather than silently dropping it.
func NormalizeTags(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	for _, r := range raw {
		t, err := NormalizeTag(r)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(out, t) {
			out = append(out, t)
		}
	}
	if len(out) > MaxTagsPerIssue {
		return nil, fmt.Errorf("%d tags, max %d: %w", len(out), MaxTagsPerIssue, ErrTooManyTags)
	}
	slices.Sort(out)
	return out, nil
}

func normalizeIssueKey(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}
