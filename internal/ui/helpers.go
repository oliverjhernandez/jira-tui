// Package ui
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderKeyBind - Helper to render a keybind in status bar: "q quit"
func RenderKeyBind(key, desc string) string {
	return StatusBarKeyStyle.Render(key) + " " + StatusBarDescStyle.Render(desc)
}

// RenderSeparator - to create a separator line
func RenderSeparator(width int) string {
	return SeparatorStyle.Render(lipgloss.NewStyle().Width(width).Render(RepeatChar("─", width)))
}

func RepeatChar(char string, count int) string {
	var result strings.Builder
	for range count {
		result.WriteString(char)
	}
	return result.String()
}

func RenderFieldStyled(label, value string, width int) string {
	style := lipgloss.NewStyle().Width(width)
	content := DetailLabelStyle.Render(label+": ") + DetailValueStyle.Render(value)
	return style.Render(content)
}

func RenderStatusBadge(status string) string {
	if strings.ToLower(status) == "selected for development" {
		status = "To Do"
	}
	statusLower := strings.ToLower(status)

	switch {
	case strings.Contains(statusLower, "trabajando"), strings.Contains(statusLower, "progress"):
		return StatusInProgressStyle.Render(IconStatusInProgress + " " + status)
	case strings.Contains(statusLower, "done"), strings.Contains(statusLower, "closed"):
		return StatusDoneStyle.Render(IconStatusDone + " " + status)
	case strings.Contains(statusLower, "blocked"):
		return StatusBlockedStyle.Render(IconStatusBlocked + " " + status)
	case strings.Contains(statusLower, "to do"):
		return StatusToDoStyle.Render(IconStatusToDo + " " + status)
	case strings.Contains(statusLower, "backlog"):
		return StatusToDoStyle.Render(IconStatusToDo + " " + status)
	case strings.Contains(statusLower, "validación"):
		return StatusValidationStyle.Render(IconStatusValidation + " " + status)
	default:
		return StatusDefaultStyle.Render("● " + status)
	}
}

// RenderPriority renders priority with icon and color
func RenderPriority(priority string, showText bool) string {
	p := strings.ToLower(priority)
	var style lipgloss.Style
	var icon string

	switch {
	case strings.Contains(p, "highest"):
		style = PriorityHighestStyle
		icon = IconPriorityHighest
	case strings.Contains(p, "high"):
		style = PriorityHighStyle
		icon = IconPriorityHigh
	case strings.Contains(p, "medium"):
		style = PriorityMediumStyle
		icon = IconPriorityMedium
	case strings.Contains(p, "low") && !strings.Contains(p, "lowest"):
		style = PriorityLowStyle
		icon = IconPriorityLow
	case strings.Contains(p, "lowest"):
		style = PriorityLowestStyle
		icon = IconPriorityLowest
	default:
		style = PriorityMediumStyle
		icon = IconPriorityMedium
	}

	if showText {
		return style.Render(icon + " " + priority)
	} else {
		return style.Render(icon)
	}
}

// RenderIssueType renders issue type with icon
func RenderIssueType(issueType string, showText bool) string {
	t := strings.ToLower(issueType)
	var style lipgloss.Style
	var icon string

	switch {
	case strings.Contains(t, "bug"):
		style = TypeBugStyle
		icon = IconBug
	case strings.Contains(t, "task"):
		style = TypeTaskStyle
		icon = IconTask
	case strings.Contains(t, "story"):
		style = TypeStoryStyle
		icon = IconStory
	case strings.Contains(t, "epic"):
		style = TypeEpicStyle
		icon = IconEpic
	case strings.Contains(t, "investigación"):
		style = TypeInvestStyle
		icon = IconInvestigacion
	case strings.Contains(t, "subtask"), strings.Contains(t, "sub-task"):
		style = TypeSubTaskStyle
		icon = IconSubtask
	default:
		style = TypeBaseStyle
		icon = IconDefault
	}

	if showText {
		return style.Render(icon + " " + issueType)
	} else {
		return style.Render(icon)
	}
}
