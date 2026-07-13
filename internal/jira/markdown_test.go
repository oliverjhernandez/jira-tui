package jira

import (
	"strings"
	"testing"
)

// findNode returns the first node of the given type in a depth-first walk.
func findNode(nodes []ContentNode, typ string) *ContentNode {
	for i := range nodes {
		if nodes[i].Type == typ {
			return &nodes[i]
		}
		if got := findNode(nodes[i].Content, typ); got != nil {
			return got
		}
	}
	return nil
}

func TestMarkdownToADFHeading(t *testing.T) {
	doc := MarkdownToADF("## Title here")
	h := findNode(doc.Content, "heading")
	if h == nil {
		t.Fatalf("no heading node: %+v", doc.Content)
	}
	if h.Attrs == nil || h.Attrs.Level != 2 {
		t.Errorf("heading level = %+v, want 2", h.Attrs)
	}
	if txt := findNode(h.Content, "text"); txt == nil || txt.Text != "Title here" {
		t.Errorf("heading text = %+v", txt)
	}
}

func TestMarkdownToADFBulletList(t *testing.T) {
	doc := MarkdownToADF("- one\n- two")
	list := findNode(doc.Content, "bulletList")
	if list == nil {
		t.Fatalf("no bulletList: %+v", doc.Content)
	}
	items := 0
	for _, c := range list.Content {
		if c.Type == "listItem" {
			items++
		}
	}
	if items != 2 {
		t.Errorf("listItems = %d, want 2", items)
	}
}

func TestMarkdownToADFOrderedList(t *testing.T) {
	doc := MarkdownToADF("1. first\n2. second")
	if findNode(doc.Content, "orderedList") == nil {
		t.Errorf("no orderedList: %+v", doc.Content)
	}
}

func TestMarkdownToADFLinkAndEmphasis(t *testing.T) {
	doc := MarkdownToADF("see **bold** and [docs](https://example.com)")

	// bold text carries a "strong" mark
	var sawStrong, sawLink bool
	var walk func(nodes []ContentNode)
	walk = func(nodes []ContentNode) {
		for _, n := range nodes {
			for _, m := range n.Marks {
				if m.Type == "strong" {
					sawStrong = true
				}
				if m.Type == "link" && m.Attrs != nil && m.Attrs.Href == "https://example.com" {
					sawLink = true
				}
			}
			walk(n.Content)
		}
	}
	walk(doc.Content)
	if !sawStrong {
		t.Error("expected a strong mark")
	}
	if !sawLink {
		t.Error("expected a link mark with the href")
	}
}

func TestMarkdownToADFCodeBlock(t *testing.T) {
	doc := MarkdownToADF("```\nfmt.Println(1)\n```")
	cb := findNode(doc.Content, "codeBlock")
	if cb == nil {
		t.Fatalf("no codeBlock: %+v", doc.Content)
	}
	if txt := findNode(cb.Content, "text"); txt == nil || !strings.Contains(txt.Text, "fmt.Println(1)") {
		t.Errorf("codeBlock text = %+v", txt)
	}
}

func TestMarkdownEmptyProducesParagraph(t *testing.T) {
	doc := MarkdownToADF("")
	if len(doc.Content) != 1 || doc.Content[0].Type != "paragraph" {
		t.Errorf("empty markdown should yield one paragraph, got %+v", doc.Content)
	}
}

func TestRoundTripStable(t *testing.T) {
	// MD -> ADF -> MD -> ADF should be stable at the ADF level.
	inputs := []string{
		"# Heading\n\nsome **bold** and *italic* text",
		"- a\n- b\n- c",
		"1. one\n2. two",
		"a [link](https://example.com) here",
		"plain paragraph",
	}
	for _, in := range inputs {
		md1 := ADFToMarkdown(MarkdownToADF(in))
		md2 := ADFToMarkdown(MarkdownToADF(md1))
		if md1 != md2 {
			t.Errorf("round trip not stable for %q:\n first: %q\nsecond: %q", in, md1, md2)
		}
		if strings.TrimSpace(md1) == "" {
			t.Errorf("round trip lost all content for %q", in)
		}
	}
}

func TestMentionRoundTrip(t *testing.T) {
	users := []User{{ID: "acc-9", Name: "Jane Doe"}}
	doc := CommentToADF("hi @[Jane Doe] there", users)

	m := findNode(doc.Content, "mention")
	if m == nil || m.Attrs == nil || m.Attrs.ID != "acc-9" {
		t.Fatalf("expected mention with id acc-9, got %+v", m)
	}
	if md := ADFToMarkdown(doc); !strings.Contains(md, "@[Jane Doe]") {
		t.Errorf("mention did not round-trip to markdown: %q", md)
	}
}
