package store

import (
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestNormalizeTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      string
		want    string
		wantErr error
	}{
		{name: "trims and lowercases", in: "  Bug  ", want: "bug"},
		{name: "collapses whitespace", in: "NEEDS REVIEW", want: "needs-review"},
		{name: "collapses runs of whitespace", in: "needs\t\n  review", want: "needs-review"},
		{name: "strips the sigil users type", in: "#urgent", want: "urgent"},
		{name: "keeps the allowed charset", in: "a/b_c.d-e", want: "a/b_c.d-e"},
		{name: "keeps digits", in: "api-v2", want: "api-v2"},
		{name: "max length is accepted", in: strings.Repeat("a", MaxTagLen), want: strings.Repeat("a", MaxTagLen)},

		{name: "empty", in: "", wantErr: ErrEmptyTag},
		{name: "only spaces", in: "   ", wantErr: ErrEmptyTag},
		{name: "bare sigil", in: "#", wantErr: ErrEmptyTag},
		{name: "non-ascii", in: "über", wantErr: ErrInvalidTag},
		{name: "punctuation", in: "tag!", wantErr: ErrInvalidTag},
		{name: "leading dash", in: "-lead", wantErr: ErrInvalidTag},
		{name: "leading dot", in: ".dot", wantErr: ErrInvalidTag},
		{name: "one over max length", in: strings.Repeat("a", MaxTagLen+1), wantErr: ErrTagTooLong},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeTag(tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NormalizeTag(%q) error = %v, want %v", tt.in, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeTag(%q) unexpected error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("NormalizeTag(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalizeTags(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		in      []string
		want    []string
		wantErr error
	}{
		{name: "sorts and dedupes across spellings", in: []string{"Bug", "api", "bug"}, want: []string{"api", "bug"}},
		{name: "empty input", in: nil, want: []string{}},
		{name: "at the cap", in: repeatedTags(MaxTagsPerIssue), want: repeatedTags(MaxTagsPerIssue)},
		{name: "over the cap", in: repeatedTags(MaxTagsPerIssue + 1), wantErr: ErrTooManyTags},
		{name: "reports a bad tag instead of dropping it", in: []string{"ok", "bad!"}, wantErr: ErrInvalidTag},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := NormalizeTags(tt.in)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("NormalizeTags(%v) error = %v, want %v", tt.in, err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("NormalizeTags(%v) unexpected error: %v", tt.in, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NormalizeTags(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

func repeatedTags(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = string(rune('a'+i)) + "tag"
	}
	return out
}
