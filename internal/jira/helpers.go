package jira

import "github.com/oliverjhernandez/jira-tui/internal/ui"

func extractText(doc *descriptionDoc) string {
	if doc == nil {
		return ""
	}

	var text string
	for _, block := range doc.Content {
		text += extractBlockText(block) + "\n"
	}
	return text
}

func extractBlockText(block contentBlock) string {
	if block.Text != "" {
		return block.Text
	}

	var text string
	for _, node := range block.Content {
		if node.Type == "mention" && node.Attrs.Text != "" {
			text += ui.MentionStyle.Render(node.Attrs.Text)
		}

		if node.Text != "" {
			text += node.Text
		}
	}
	return text
}
