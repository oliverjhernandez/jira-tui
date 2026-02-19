package jira

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/ansi"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func ExtractText(doc *contentDoc, panelWidth int) string {
	if doc == nil {
		return ""
	}

	var text strings.Builder
	for _, block := range doc.Content {
		text.WriteString(extractBlockText(block, panelWidth) + "\n")
	}
	return text.String()
}

func extractBlockText(block contentBlock, panelWidth int) string {
	switch block.Type {
	case "heading":
		return formatHeading(block)
	case "paragraph":
		return formatParagraph(block)
	case "codeBlock":
		return formatCodeBlock(block)
	case "bulletList":
		return formatBulletList(block)
	case "orderedList":
		return formatOrderedList(block)
	case "table":
		return formatTable(block, panelWidth)
	case "rule":
		return "─────────────────────"
	default:
		var text strings.Builder
		for _, node := range block.Content {
			text.WriteString(extractInlineText(node))
		}
		return text.String()
	}
}

func extractInlineText(node contentNode) string {
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

func formatParagraph(block contentBlock) string {
	var text strings.Builder
	for _, node := range block.Content {
		text.WriteString(extractInlineText(node))
	}
	return text.String()
}

func formatHeading(block contentBlock) string {
	var text strings.Builder
	for _, node := range block.Content {
		text.WriteString(extractInlineText(node))
	}
	return ui.HeadingStyle.Render("# " + text.String())
}

func formatCodeBlock(block contentBlock) string {
	var text strings.Builder
	for _, node := range block.Content {
		text.WriteString(extractInlineText(node))
	}
	return ui.CodeBlockStyle.Render(text.String())
}

func formatOrderedList(block contentBlock) string {
	var items strings.Builder
	for i, item := range block.Content {
		if item.Type == "listItem" {
			itemText := extractListItemText(item)
			fmt.Fprintf(&items, "  %d. %s\n", i+1, itemText)
		}
	}
	return items.String()
}

func formatBulletList(block contentBlock) string {
	var items strings.Builder
	for _, item := range block.Content {
		if item.Type == "listItem" {
			itemText := formatListItem(item)
			items.WriteString("  • " + itemText + "\n")
		}
	}
	return items.String()
}

func formatListItem(node contentNode) string {
	var text strings.Builder
	for _, child := range node.Content {
		if child.Type == "paragraph" {
			text.WriteString(formatParagraph(contentBlock{Content: child.Content}))
		}
	}
	return text.String()
}

func extractListItemText(item contentNode) string {
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

func formatTable(block contentBlock, panelWidth int) string {
	var allRows [][]string

	for _, row := range block.Content {
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
		for i, cell := range row {
			cellStyle := lipgloss.NewStyle().Width(colWidths[i])
			padded := cellStyle.Render(cell)
			output.WriteString(padded)
		}
		output.WriteString("\n")
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
				cellWidth := ansi.PrintableRuneWidth(cell)
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

func extractRowCells(row contentNode) []string {
	var cells []string

	for _, cell := range row.Content {
		if cell.Type == "tableHeader" || cell.Type == "tableCell" {
			cellText := extractTableCellText(cell)
			cells = append(cells, cellText)
		}
	}

	return cells
}

func extractTableCellText(cell contentNode) string {
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

func formatSecondsToTime(seconds int) string {
	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return ""
}
