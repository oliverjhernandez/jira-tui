package jira

import (
	"strings"

	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

func extractText(doc *descriptionDoc) string {
	if doc == nil {
		return ""
	}

	var text strings.Builder
	for _, block := range doc.Content {
		text.WriteString(extractBlockText(block) + "\n")
	}
	return text.String()
}

func extractBlockText(block contentBlock) string {
	if block.Text != "" {
		return block.Text
	}

	var text strings.Builder
	for _, node := range block.Content {
		if node.Type == "mention" && node.Attrs.Text != "" {
			text.WriteString(ui.MentionStyle.Render(node.Attrs.Text))
		}

		if node.Text != "" {
			text.WriteString(node.Text)
		}
	}
	return text.String()
}
