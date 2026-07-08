package jira

import (
	"strings"
	"testing"
)

func TestExtractTextNil(t *testing.T) {
	if got := ExtractText(nil, 80); got != "" {
		t.Errorf("ExtractText(nil) = %q, want empty", got)
	}
}

func TestExtractTextParagraph(t *testing.T) {
	doc := &ContentDoc{
		Type: "doc",
		Content: []ContentNode{
			{
				Type: "paragraph",
				Content: []ContentNode{
					{Type: "text", Text: "Hello "},
					{Type: "text", Text: "world"},
				},
			},
		},
	}
	got := ExtractText(doc, 80)
	if !strings.Contains(got, "Hello") || !strings.Contains(got, "world") {
		t.Errorf("ExtractText paragraph = %q, want it to contain Hello world", got)
	}
}

func TestExtractTextBlockTypes(t *testing.T) {
	types := []string{"heading", "codeBlock", "bulletList", "orderedList", "rule"}
	for _, typ := range types {
		t.Run(typ, func(t *testing.T) {
			doc := &ContentDoc{
				Content: []ContentNode{
					{
						Type: typ,
						Content: []ContentNode{
							{Type: "text", Text: "content"},
						},
					},
				},
			}
			// Should not panic for any supported block type.
			_ = ExtractText(doc, 80)
		})
	}
}

func TestExtractTextMentionNilAttrs(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ExtractText panicked on mention with nil Attrs: %v", r)
		}
	}()
	doc := &ContentDoc{
		Content: []ContentNode{
			{
				Type: "paragraph",
				Content: []ContentNode{
					{Type: "mention"}, // Attrs is nil
				},
			},
		},
	}
	_ = ExtractText(doc, 80)
}

func TestExtractTextMediaNilAttrs(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ExtractText panicked on media with nil Attrs: %v", r)
		}
	}()
	doc := &ContentDoc{
		Content: []ContentNode{
			{
				Type: "mediaSingle",
				Content: []ContentNode{
					{Type: "media"}, // Attrs is nil
				},
			},
		},
	}
	_ = ExtractText(doc, 80)
}

func TestExtractTextIrregularTable(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("ExtractText panicked on table with irregular rows: %v", r)
		}
	}()

	cell := func(text string) ContentNode {
		return ContentNode{
			Type: "tableCell",
			Content: []ContentNode{
				{Type: "paragraph", Content: []ContentNode{{Type: "text", Text: text}}},
			},
		}
	}
	row := func(cells ...ContentNode) ContentNode {
		return ContentNode{Type: "tableRow", Content: cells}
	}

	doc := &ContentDoc{
		Content: []ContentNode{
			{
				Type: "table",
				Content: []ContentNode{
					row(cell("a"), cell("b")),            // 2 cells
					row(cell("c"), cell("d"), cell("e")), // 3 cells (more than first row)
				},
			},
		},
	}
	_ = ExtractText(doc, 120)
}

func TestHardWrap(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
		want  string
	}{
		{"no wrap needed", "abc", 10, "abc"},
		{"wrap long line", "abcdef", 3, "abc\ndef"},
		{"zero width returns input", "abcdef", 0, "abcdef"},
		{"negative width returns input", "abcdef", -1, "abcdef"},
		{"preserves existing newlines", "ab\ncd", 10, "ab\ncd"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hardWrap(tt.in, tt.width); got != tt.want {
				t.Errorf("hardWrap(%q, %d) = %q, want %q", tt.in, tt.width, got, tt.want)
			}
		})
	}
}

func TestParseCommentContentNoMention(t *testing.T) {
	content, err := parseCommentContent("just a plain comment", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(content) != 1 {
		t.Fatalf("expected 1 node, got %d", len(content))
	}
	if content[0]["type"] != "text" || content[0]["text"] != "just a plain comment" {
		t.Errorf("unexpected node: %+v", content[0])
	}
}

func TestParseCommentContentWithKnownMention(t *testing.T) {
	users := []User{{ID: "acc-1", Name: "Jane Doe"}}
	content, err := parseCommentContent("hi @[Jane Doe] bye", users)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var sawMention bool
	for _, node := range content {
		if node["type"] == "mention" {
			sawMention = true
			attrs, ok := node["attrs"].(map[string]string)
			if !ok {
				t.Fatalf("mention attrs wrong type: %T", node["attrs"])
			}
			if attrs["id"] != "acc-1" {
				t.Errorf("mention id = %q, want acc-1", attrs["id"])
			}
		}
	}
	if !sawMention {
		t.Errorf("expected a mention node, got %+v", content)
	}
}

func TestParseCommentContentUnknownMentionStaysText(t *testing.T) {
	content, err := parseCommentContent("hi @[Nobody Here]", []User{{ID: "x", Name: "Someone Else"}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, node := range content {
		if node["type"] == "mention" {
			t.Errorf("did not expect a mention node for an unknown user: %+v", content)
		}
	}
}

func TestLinkTypeString(t *testing.T) {
	cases := map[LinkType]string{
		Relates:       "Relates",
		Blocks:        "Blocks",
		Duplicates:    "Duplicates",
		LinkType(999): "",
	}
	for lt, want := range cases {
		if got := lt.String(); got != want {
			t.Errorf("LinkType(%d).String() = %q, want %q", lt, got, want)
		}
	}
}
