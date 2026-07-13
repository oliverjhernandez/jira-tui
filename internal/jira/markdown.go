package jira

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// mentionRegex matches the "@[Display Name]" mention syntax used in comments.
var mentionRegex = regexp.MustCompile(`@\[([^\]]+)\]`)

// mdParser is a CommonMark parser (no extensions) shared across conversions.
var mdParser = goldmark.New()

// MarkdownToADF converts Markdown source into a Jira ADF document. It supports
// the subset the TUI can round-trip: headings, paragraphs, bold/italic/inline
// code, links, bullet/ordered lists (incl. nesting), fenced code blocks and
// horizontal rules. Anything else degrades to its plain text.
func MarkdownToADF(src string) *ContentDoc {
	source := []byte(src)
	root := mdParser.Parser().Parse(text.NewReader(source))

	doc := &ContentDoc{Type: "doc", Version: 1}
	doc.Content = blocksToNodes(root, source)
	if len(doc.Content) == 0 {
		// ADF requires at least one block node.
		doc.Content = []ContentNode{{Type: "paragraph"}}
	}
	return doc
}

// blocksToNodes converts every block-level child of parent into ADF nodes.
func blocksToNodes(parent ast.Node, source []byte) []ContentNode {
	var out []ContentNode
	for c := parent.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, blockToNodes(c, source)...)
	}
	return out
}

func blockToNodes(n ast.Node, source []byte) []ContentNode {
	switch node := n.(type) {
	case *ast.Heading:
		return []ContentNode{{
			Type:    "heading",
			Attrs:   &contentAttrs{Level: node.Level},
			Content: inlineChildren(n, source),
		}}
	case *ast.Paragraph, *ast.TextBlock:
		return []ContentNode{{Type: "paragraph", Content: inlineChildren(n, source)}}
	case *ast.List:
		listType := "bulletList"
		if node.IsOrdered() {
			listType = "orderedList"
		}
		var items []ContentNode
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if _, ok := c.(*ast.ListItem); ok {
				items = append(items, ContentNode{Type: "listItem", Content: blocksToNodes(c, source)})
			}
		}
		return []ContentNode{{Type: listType, Content: items}}
	case *ast.FencedCodeBlock:
		return []ContentNode{codeBlockNode(node.Lines(), source)}
	case *ast.CodeBlock:
		return []ContentNode{codeBlockNode(node.Lines(), source)}
	case *ast.ThematicBreak:
		return []ContentNode{{Type: "rule"}}
	case *ast.Blockquote:
		// The renderer has no blockquote; flatten its inner blocks so they still
		// display and round-trip rather than showing "Unknown node type".
		return blocksToNodes(n, source)
	default:
		// Unknown block: keep any inline text as a paragraph so nothing is lost.
		if inline := inlineChildren(n, source); len(inline) > 0 {
			return []ContentNode{{Type: "paragraph", Content: inline}}
		}
		return nil
	}
}

func codeBlockNode(lines *text.Segments, source []byte) ContentNode {
	var b strings.Builder
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		b.Write(seg.Value(source))
	}
	code := strings.TrimRight(b.String(), "\n")
	return ContentNode{Type: "codeBlock", Content: []ContentNode{{Type: "text", Text: code}}}
}

func inlineChildren(n ast.Node, source []byte) []ContentNode {
	var out []ContentNode
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, inlineToNodes(c, source, nil)...)
	}
	return out
}

// inlineToNodes converts an inline node, carrying accumulated marks from any
// enclosing emphasis/link/code spans.
func inlineToNodes(n ast.Node, source []byte, marks []mark) []ContentNode {
	switch node := n.(type) {
	case *ast.Text:
		var out []ContentNode
		if seg := string(node.Segment.Value(source)); seg != "" {
			out = append(out, ContentNode{Type: "text", Text: seg, Marks: marks})
		}
		if node.HardLineBreak() {
			out = append(out, ContentNode{Type: "hardBreak"})
		} else if node.SoftLineBreak() {
			out = append(out, ContentNode{Type: "text", Text: " ", Marks: marks})
		}
		return out
	case *ast.String:
		return []ContentNode{{Type: "text", Text: string(node.Value), Marks: marks}}
	case *ast.CodeSpan:
		var sb strings.Builder
		for c := n.FirstChild(); c != nil; c = c.NextSibling() {
			if t, ok := c.(*ast.Text); ok {
				sb.Write(t.Segment.Value(source))
			}
		}
		return []ContentNode{{Type: "text", Text: sb.String(), Marks: addMark(marks, mark{Type: "code"})}}
	case *ast.Emphasis:
		markType := "em"
		if node.Level == 2 {
			markType = "strong"
		}
		return inlineChildrenWithMarks(n, source, addMark(marks, mark{Type: markType}))
	case *ast.Link:
		href := string(node.Destination)
		return inlineChildrenWithMarks(n, source, addMark(marks, mark{Type: "link", Attrs: &markAttrs{Href: href}}))
	case *ast.AutoLink:
		url := string(node.URL(source))
		return []ContentNode{{Type: "text", Text: url, Marks: addMark(marks, mark{Type: "link", Attrs: &markAttrs{Href: url}})}}
	default:
		return inlineChildrenWithMarks(n, source, marks)
	}
}

func inlineChildrenWithMarks(n ast.Node, source []byte, marks []mark) []ContentNode {
	var out []ContentNode
	for c := n.FirstChild(); c != nil; c = c.NextSibling() {
		out = append(out, inlineToNodes(c, source, marks)...)
	}
	return out
}

func addMark(marks []mark, m mark) []mark {
	next := make([]mark, len(marks), len(marks)+1)
	copy(next, marks)
	return append(next, m)
}

// ADFToMarkdown converts an ADF document back into Markdown for pre-filling the
// edit form, over the same subset MarkdownToADF supports. Nodes outside that
// subset degrade to their plain text (see ADFHasUnsupported to warn the user).
func ADFToMarkdown(doc *ContentDoc) string {
	if doc == nil {
		return ""
	}
	var b strings.Builder
	for _, node := range doc.Content {
		blockToMarkdown(node, 0, &b)
	}
	return strings.TrimSpace(b.String())
}

func blockToMarkdown(node ContentNode, indent int, b *strings.Builder) {
	switch node.Type {
	case "paragraph":
		b.WriteString(inlineToMarkdown(node.Content) + "\n\n")
	case "heading":
		level := 1
		if node.Attrs != nil && node.Attrs.Level > 0 {
			level = node.Attrs.Level
		}
		b.WriteString(strings.Repeat("#", level) + " " + inlineToMarkdown(node.Content) + "\n\n")
	case "bulletList":
		listToMarkdown(node, false, indent, b)
		if indent == 0 {
			b.WriteString("\n")
		}
	case "orderedList":
		listToMarkdown(node, true, indent, b)
		if indent == 0 {
			b.WriteString("\n")
		}
	case "codeBlock":
		b.WriteString("```\n" + inlineToMarkdown(node.Content) + "\n```\n\n")
	case "rule":
		b.WriteString("---\n\n")
	default:
		if s := inlineToMarkdown(node.Content); s != "" {
			b.WriteString(s + "\n\n")
		}
	}
}

func listToMarkdown(list ContentNode, ordered bool, indent int, b *strings.Builder) {
	pad := strings.Repeat("  ", indent)
	n := 1
	for _, item := range list.Content {
		if item.Type != "listItem" {
			continue
		}
		marker := "- "
		if ordered {
			marker = fmt.Sprintf("%d. ", n)
		}
		wroteMarker := false
		for _, child := range item.Content {
			switch child.Type {
			case "bulletList":
				listToMarkdown(child, false, indent+1, b)
			case "orderedList":
				listToMarkdown(child, true, indent+1, b)
			default:
				line := inlineToMarkdown(child.Content)
				if !wroteMarker {
					b.WriteString(pad + marker + line + "\n")
					wroteMarker = true
				} else {
					b.WriteString(pad + "  " + line + "\n")
				}
			}
		}
		if !wroteMarker {
			b.WriteString(pad + marker + "\n")
		}
		n++
	}
}

func inlineToMarkdown(nodes []ContentNode) string {
	var b strings.Builder
	for _, n := range nodes {
		switch n.Type {
		case "text":
			b.WriteString(applyInlineMarks(n.Text, n.Marks))
		case "mention":
			if n.Attrs != nil {
				b.WriteString("@[" + strings.TrimPrefix(n.Attrs.Text, "@") + "]")
			}
		case "hardBreak":
			b.WriteString("\n")
		case "inlineCard":
			if n.Attrs != nil {
				b.WriteString(n.Attrs.URL)
			}
		default:
			b.WriteString(inlineToMarkdown(n.Content))
		}
	}
	return b.String()
}

func applyInlineMarks(s string, marks []mark) string {
	if s == "" {
		return s
	}
	var href string
	link := false
	for _, m := range marks {
		switch m.Type {
		case "code":
			s = "`" + s + "`"
		case "em":
			s = "*" + s + "*"
		case "strong":
			s = "**" + s + "**"
		case "link":
			link = true
			if m.Attrs != nil {
				href = m.Attrs.Href
			}
		}
	}
	if link {
		s = "[" + s + "](" + href + ")"
	}
	return s
}

// ADFHasUnsupported reports whether the document contains block types that the
// Markdown round-trip cannot preserve (tables, media, panels, etc.), so callers
// can warn before an edit silently drops them.
func ADFHasUnsupported(doc *ContentDoc) bool {
	if doc == nil {
		return false
	}
	supported := map[string]bool{
		"paragraph": true, "heading": true, "bulletList": true, "orderedList": true,
		"listItem": true, "codeBlock": true, "rule": true, "blockquote": true,
	}
	for _, n := range doc.Content {
		if !supported[n.Type] {
			return true
		}
	}
	return false
}

// Private-use sentinels wrap a mention index while the text passes through the
// Markdown parser, so goldmark can't reinterpret the "@[Name]" brackets as link
// syntax. They are ordinary letters to the parser and survive intact in a single
// text node, where restoreMentions swaps them back.
const (
	mentionOpen  = ""
	mentionClose = ""
)

var mentionPlaceholder = regexp.MustCompile(mentionOpen + `(\d+)` + mentionClose)

// CommentToADF converts a Markdown comment to ADF while turning "@[Name]"
// mentions (resolved against users) into ADF mention nodes.
func CommentToADF(src string, users []User) *ContentDoc {
	protected, names := protectMentions(src)
	doc := MarkdownToADF(protected)
	doc.Content = restoreMentions(doc.Content, names, users)
	return doc
}

// protectMentions replaces each "@[Name]" with an index sentinel and returns
// the names in order.
func protectMentions(src string) (string, []string) {
	var names []string
	out := mentionRegex.ReplaceAllStringFunc(src, func(match string) string {
		name := mentionRegex.FindStringSubmatch(match)[1]
		idx := len(names)
		names = append(names, name)
		return fmt.Sprintf("%s%d%s", mentionOpen, idx, mentionClose)
	})
	return out, names
}

// restoreMentions walks ADF text nodes and swaps mention sentinels for ADF
// mention nodes, resolving each name to an account id. Unknown names fall back
// to literal "@[Name]" text.
func restoreMentions(nodes []ContentNode, names []string, users []User) []ContentNode {
	var out []ContentNode
	for _, n := range nodes {
		if n.Type == "text" && mentionPlaceholder.MatchString(n.Text) {
			out = append(out, splitPlaceholders(n, names, users)...)
			continue
		}
		if len(n.Content) > 0 {
			n.Content = restoreMentions(n.Content, names, users)
		}
		out = append(out, n)
	}
	return out
}

func splitPlaceholders(node ContentNode, names []string, users []User) []ContentNode {
	matches := mentionPlaceholder.FindAllStringSubmatchIndex(node.Text, -1)
	var out []ContentNode
	last := 0
	for _, m := range matches {
		start, end := m[0], m[1]
		idx, _ := strconv.Atoi(node.Text[m[2]:m[3]])
		if start > last {
			out = append(out, ContentNode{Type: "text", Text: node.Text[last:start], Marks: node.Marks})
		}
		name := ""
		if idx >= 0 && idx < len(names) {
			name = names[idx]
		}
		accountID := ""
		for _, u := range users {
			if u.Name == name {
				accountID = u.ID
				break
			}
		}
		if accountID == "" {
			out = append(out, ContentNode{Type: "text", Text: "@[" + name + "]", Marks: node.Marks})
		} else {
			out = append(out, ContentNode{Type: "mention", Attrs: &contentAttrs{ID: accountID, Text: "@" + name}})
		}
		last = end
	}
	if last < len(node.Text) {
		out = append(out, ContentNode{Type: "text", Text: node.Text[last:], Marks: node.Marks})
	}
	return out
}
