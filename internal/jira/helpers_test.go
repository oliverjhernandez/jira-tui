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

func TestCommentToADFNoMention(t *testing.T) {
	doc := CommentToADF("just a plain comment", nil)
	if txt := findNode(doc.Content, "text"); txt == nil || txt.Text != "just a plain comment" {
		t.Errorf("unexpected text node: %+v", txt)
	}
	if findNode(doc.Content, "mention") != nil {
		t.Error("did not expect a mention node")
	}
}

func TestCommentToADFWithKnownMention(t *testing.T) {
	users := []User{{ID: "acc-1", Name: "Jane Doe"}}
	doc := CommentToADF("hi @[Jane Doe] bye", users)

	m := findNode(doc.Content, "mention")
	if m == nil {
		t.Fatalf("expected a mention node, got %+v", doc.Content)
	}
	if m.Attrs == nil || m.Attrs.ID != "acc-1" {
		t.Errorf("mention id = %+v, want acc-1", m.Attrs)
	}
	if m.Attrs != nil && m.Attrs.Text != "@Jane Doe" {
		t.Errorf("mention text = %q, want @Jane Doe", m.Attrs.Text)
	}
}

func TestCommentToADFUnknownStaysText(t *testing.T) {
	doc := CommentToADF("hi @[Nobody Here]", []User{{ID: "x", Name: "Someone Else"}})
	if findNode(doc.Content, "mention") != nil {
		t.Errorf("did not expect a mention node for an unknown user: %+v", doc.Content)
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
