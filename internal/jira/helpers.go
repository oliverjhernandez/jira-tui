package jira

import (
	"fmt"
	"regexp"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func ExtractText(doc *ContentDoc, panelWidth int) string {
	if doc == nil {
		return ""
	}

	var text strings.Builder
	for _, node := range doc.Content {
		text.WriteString(extractBlockText(node, panelWidth-ui.PanelOverheadWidth) + "\n")
	}
	return text.String()
}

func extractBlockText(node ContentNode, panelWidth int) string {
	switch node.Type {
	case "heading":
		return formatHeading(node)
	case "paragraph":
		return formatParagraph(node, panelWidth)
	case "codeBlock":
		return formatCodeBlock(node)
	case "bulletList":
		return formatBulletList(node, 0, panelWidth)
	case "orderedList":
		return formatOrderedList(node)
	case "table":
		return formatTable(node, panelWidth)
	case "mediaSingle":
		return formatMediaSingle(node)
	case "rule":
		return "─────────────────────"
	default:
		var text strings.Builder
		text.WriteString(fmt.Sprintf("Unknown node type: %s\n", node.Type))
		for _, node := range node.Content {
			text.WriteString(extractInlineText(node))
		}
		return text.String()
	}
}

func hardWrap(s string, width int) string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		for len(line) > width {
			result = append(result, line[:width])
			line = line[width:]
		}
		result = append(result, line)
	}
	return strings.Join(result, "\n")
}

func formatParagraph(node ContentNode, panelWidth int) string {
	var text strings.Builder
	for _, node := range node.Content {
		text.WriteString(extractInlineText(node))
	}

	wrapped := ansi.Wrap(text.String(), panelWidth, "")
	return wrapped + "\n"
}

func extractInlineText(node ContentNode) string {
	if node.Type == "mention" {
		userName := node.Attrs.Text
		if userName == "" {
			userName = node.Attrs.ID
		}
		return ui.MentionStyle.Render(userName)
	} else if node.Type == "hardBreak" {
		return "\n"
	}

	text := node.Text

	for _, mark := range node.Marks {
		switch mark.Type {
		case "strong":
			text = ui.BoldStyle.Render(text)
		case "em":
			text = ui.ItalicStyle.Render(text)
		case "code":
			text = ui.InlineCodeStyle.Render(text)
		}
	}

	for _, child := range node.Content {
		text += extractInlineText(child)
	}

	return text
}

func formatHeading(node ContentNode) string {
	var text strings.Builder
	for _, node := range node.Content {
		text.WriteString(extractInlineText(node))
	}
	return ui.HeadingStyle.Render("# " + text.String())
}

func formatCodeBlock(node ContentNode) string {
	var text strings.Builder
	for _, node := range node.Content {
		text.WriteString(extractInlineText(node))
	}
	return ui.CodeBlockStyle.Render(text.String())
}

func formatOrderedList(node ContentNode) string {
	var items strings.Builder
	for i, item := range node.Content {
		if item.Type == "listItem" {
			itemText := extractListItemText(item)
			fmt.Fprintf(&items, "  %d. %s\n", i+1, itemText)
		}
	}
	return items.String()
}

func formatBulletList(node ContentNode, indent int, width int) string {
	var items strings.Builder
	for _, item := range node.Content {
		if item.Type == "listItem" {
			bullet := ui.IconBullet + " "
			itemText := formatListItem(item, indent, width-lipgloss.Width(bullet))
			items.WriteString(strings.Repeat("  ", indent) + bullet + itemText + "\n")
		}
	}
	return items.String()
}

func formatMediaSingle(node ContentNode) string {
	var content strings.Builder
	for _, item := range node.Content {
		if item.Type == "media" {
			content.WriteString("[📎 " + item.Attrs.Alt + "]\n")
		}
	}

	return content.String()
}

func formatListItem(node ContentNode, indent int, width int) string {
	var text strings.Builder
	for _, child := range node.Content {
		switch child.Type {
		case "paragraph":
			text.WriteString(formatParagraph(child, width))
		case "bulletList":
			text.WriteString(formatBulletList(child, indent+1, width))
		}
	}
	return text.String()
}

func extractListItemText(item ContentNode) string {
	var text strings.Builder
	for _, child := range item.Content {
		if child.Type == "paragraph" {
			for _, node := range child.Content {
				text.WriteString(extractInlineText(node))
			}
		}
	}
	return text.String()
}

func formatTable(node ContentNode, panelWidth int) string {
	var allRows [][]string

	for _, row := range node.Content {
		if row.Type == "tableRow" {
			cells := extractRowCells(row)
			if !isEmptyRow(cells) {
				allRows = append(allRows, cells)
			}
		}
	}

	if len(allRows) == 0 {
		return ""
	}

	colWidths := calculateColumnWidths(allRows, panelWidth)

	var output strings.Builder
	for _, row := range allRows {
		var cells []string
		for i, cell := range row {
			cellStyle := lipgloss.NewStyle().Width(colWidths[i])
			cells = append(cells, cellStyle.Render(hardWrap(cell, colWidths[i])))
		}
		output.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, cells...) + "\n")
	}

	return output.String()
}

func calculateColumnWidths(rows [][]string, maxWidth int) []int {
	if len(rows) == 0 {
		return nil
	}

	numCols := len(rows[0])
	widths := make([]int, numCols)

	for _, row := range rows {
		for i, cell := range row {
			if i < numCols {
				cellWidth := ansi.StringWidth(cell)
				if cellWidth > widths[i] {
					widths[i] = cellWidth
				}
			}
		}
	}

	for i := range widths {
		widths[i] += 4
		if widths[i] < 25 {
			widths[i] = 25
		}
	}

	totalWidth := 0
	for _, w := range widths {
		totalWidth += w
	}

	if totalWidth > maxWidth {
		scale := float64(maxWidth) / float64(totalWidth)
		for i := range widths {
			widths[i] = max(int(float64(widths[i])*scale),
				10)
		}
	} else {
		scale := float64(maxWidth) / float64(totalWidth)
		for i := range widths {
			widths[i] = int(float64(widths[i]) * scale)
		}
	}

	return widths
}

func isEmptyRow(cells []string) bool {
	for _, cell := range cells {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func extractRowCells(row ContentNode) []string {
	var cells []string

	for _, cell := range row.Content {
		if cell.Type == "tableHeader" || cell.Type == "tableCell" {
			cellText := extractTableCellText(cell)
			cells = append(cells, cellText)
		}
	}

	return cells
}

func extractTableCellText(cell ContentNode) string {
	var text strings.Builder
	for _, child := range cell.Content {
		if child.Type == "paragraph" {
			for _, node := range child.Content {
				text.WriteString(extractInlineText(node))
			}
		}
	}
	return text.String()
}

func parseCommentContent(comment string, users []User) ([]map[string]any, error) {
	mentionRegex := regexp.MustCompile(`@\[([^\]]+)\]`)

	matches := mentionRegex.FindAllStringSubmatchIndex(comment, -1)

	if len(matches) == 0 {
		return []map[string]any{
			{
				"type": "text",
				"text": comment,
			},
		}, nil
	}

	var content []map[string]any
	lastEnd := 0

	for _, match := range matches {
		matchStart := match[0]
		matchEnd := match[1]
		nameStart := match[2]
		nameEnd := match[3]

		if matchStart > lastEnd {
			content = append(content, map[string]any{
				"type": "text",
				"text": comment[lastEnd:matchStart],
			})
		}

		displayName := comment[nameStart:nameEnd]

		var accountID string
		for _, user := range users {
			if user.Name == displayName {
				accountID = user.ID
				break
			}
		}

		if accountID == "" {
			content = append(content, map[string]any{
				"type": "text",
				"text": comment[matchStart:matchEnd],
			})
		} else {
			content = append(content, map[string]any{
				"type": "mention",
				"attrs": map[string]string{
					"id":   accountID,
					"text": "@" + displayName,
				},
			})
		}

		lastEnd = matchEnd
	}

	if lastEnd < len(comment) {
		content = append(content, map[string]any{
			"type": "text",
			"text": comment[lastEnd:],
		})
	}

	return content, nil
}
