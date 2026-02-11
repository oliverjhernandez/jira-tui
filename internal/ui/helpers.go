// Package ui
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// var (
// 	modalWidth  = getModalWidth(0.8)
// 	modalHeight = getModalHeight(0.7)
// 	x, y        = getCenteredModalPosition(modalWidth, modalHeight)
// )

// FormatTimeSpent formats seconds into a human readable string like "3h" or "2h 30m"
func FormatTimeSpent(seconds int) string {
	if seconds == 0 {
		return "-"
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60

	if hours > 0 && minutes > 0 {
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm", minutes)
	}
	return "-"
}

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
	content := DetailLabelStyle.Render(label+": ") + DetailValueStyle.Render(value)

	return lipgloss.NewStyle().
		Width(width).
		MaxWidth(width).
		Inline(true).
		Render(content)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
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
	case strings.Contains(p, "crítica"):
		style = PriorityCriticalStyle
		icon = IconPriorityCritical
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
	case strings.Contains(t, "bug") || strings.Contains(t, "defecto qa"):
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

func RenderCenteredModal(content string, background string, windowWidth, windowHeight int, style lipgloss.Style) string {
	styledModal := style.Render(content)

	modalWidth := lipgloss.Width(styledModal)
	modalHeight := lipgloss.Height(styledModal)

	x := (windowWidth - modalWidth) / 2
	y := (windowHeight - modalHeight) / 2

	return placeOverlay(x, y, styledModal, background, false)
}

func GetPanelWidth(windowWidth int) int {
	return max(120, windowWidth-2)
}

func GetPanelHeight(windowHeight int) int {
	infoPanelHeight := 6
	return windowHeight - 2 - infoPanelHeight
}

func GetModalWidth(windowWidth int, scale float64) int {
	return int(float64(windowWidth) * scale)
}

func GetModalHeight(windowHeight int, scale float64) int {
	return int(float64(windowHeight) * scale)
}

// func GetLargeModalWidth(windowWidth int) int {
// 	return getModalWidth(windowWidth, 0.7)
// }

func GetListViewportWidth(windowWidth int) int {
	return windowWidth - 4
}

func GetListViewportHeight(windowHeight int) int {
	infoPanelHeight := 6
	return windowHeight - 3 - infoPanelHeight
}

func GetDetailViewportWidth(windowWidth int) int {
	return windowWidth - 10
}

func GetDetailViewportHeight(windowHeight int) int {
	headerHeight := 15
	footerHeight := 1
	return windowHeight - headerHeight - footerHeight
}
