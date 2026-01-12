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
func RenderPriority(priority string) string {
	p := strings.ToLower(priority)
	switch {
	case strings.Contains(p, "highest"):
		return PriorityHighestStyle.Render(IconPriorityHighest)
	case strings.Contains(p, "high"):
		return PriorityHighStyle.Render(IconPriorityHigh)
	case strings.Contains(p, "medium"):
		return PriorityMediumStyle.Render(IconPriorityMedium)
	case strings.Contains(p, "low") && !strings.Contains(p, "lowest"):
		return PriorityLowStyle.Render(IconPriorityLow)
	case strings.Contains(p, "lowest"):
		return PriorityLowestStyle.Render(IconPriorityLowest)
	default:
		return PriorityMediumStyle.Render(IconPriorityMedium)
	}
}

// RenderIssueType renders issue type with icon
func RenderIssueType(issueType string) string {
	t := strings.ToLower(issueType)
	style := lipgloss.NewStyle().Width(ColWidthType)

	switch {
	case strings.Contains(t, "bug"):
		return style.Foreground(ThemeError).Render(IconBug)
	case strings.Contains(t, "task"):
		return style.Foreground(ThemeInfo).Render(IconTask)
	case strings.Contains(t, "story"):
		return style.Foreground(ThemeSuccess).Render(IconStory)
	case strings.Contains(t, "epic"):
		return style.Foreground(ThemeAccentAlt).Render(IconEpic)
	case strings.Contains(t, "investigación"):
		return style.Foreground(ThemeAccentAlt).Render(IconInvestigacion)
	case strings.Contains(t, "subtask"), strings.Contains(t, "sub-task"):
		return style.Foreground(ThemeFgMuted).Render(IconSubtask)
	default:
		return style.Render(IconDefault)
	}
}
