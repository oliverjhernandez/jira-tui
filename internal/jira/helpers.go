package jira

import (
	"fmt"
	"regexp"
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
