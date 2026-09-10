// Package ui
package ui

import (
	"fmt"
	"image/color"
	"strings"
	"time"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

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

// // RenderKeyBind - Helper to render a keybind in status bar: "q quit"
// func RenderKeyBind(key, desc string) string {
// 	return StatusBarKeyStyle.Render(key) + " " + StatusBarDescStyle.Render(desc)
// }

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

// func truncate(s string, maxLen int) string {
// 	if len(s) <= maxLen {
// 		return s
// 	}
// 	return s[:maxLen]
// }

func RenderStatusBadge(status string) string {
	if strings.ToLower(status) == "selected for development" {
		status = "Selected"
	}

	if strings.ToLower(status) == "ready to deploy" {
		status = "Ready"
	}
	statusLower := strings.ToLower(status)

	switch {
	case strings.Contains(statusLower, "trabajando"), strings.Contains(statusLower, "progress"):
		return StatusInProgressStyle.Render(IconStatusInProgress + " " + status)
	case strings.Contains(statusLower, "done"), strings.Contains(statusLower, "closed"):
		return StatusDoneStyle.Render(IconStatusDone + " " + status)
	case strings.Contains(statusLower, "ready to deploy"), strings.Contains(statusLower, "ready"):
		return StatusReadyStyle.Render(IconStatusReady + " " + status)
	case strings.Contains(statusLower, "blocked"):
		return StatusBlockedStyle.Render(IconStatusBlocked + " " + status)
	case strings.Contains(statusLower, "to do"):
		return StatusToDoStyle.Render(IconStatusToDo + " " + status)
	case strings.Contains(statusLower, "backlog"):
		return StatusToDoStyle.Render(IconStatusBacklog + " " + status)
	case strings.Contains(statusLower, "validación"):
		return StatusValidationStyle.Render(IconStatusValidation + " " + status)
	case strings.Contains(statusLower, "selected"):
		return StatusSelectedStyle.Render(IconStatusSelected + " " + status)

	default:
		return StatusDefaultStyle.Render(IconSeparator + " " + status)
	}
}

// RenderPriority renders priority with icon and color
func RenderPriority(priority string, showText bool) string {
	p := strings.ToLower(strings.TrimSpace(priority))
	var style lipgloss.Style
	var icon string

	switch {
	case p == "", strings.Contains(p, "definir"), p == "none":
		style = PriorityUnsetStyle
		icon = IconPriorityUnset
	case strings.Contains(p, "critica") || strings.Contains(p, "crítica"):
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
		style = PriorityUnsetStyle
		icon = IconPriorityUnset
	}

	if showText && p != "" {
		return style.Render(icon + " " + priority)
	}
	return style.Render(icon)
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
	case strings.Contains(t, "children"), strings.Contains(t, "sub-task"):
		style = TypeSubtaskStyle
		icon = IconSubTask
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

func GetModalWidth(windowWidth int, scale float64) int {
	return int(float64(windowWidth) * scale)
}

func GetModalHeight(windowHeight int, scale float64) int {
	return int(float64(windowHeight) * scale)
}

func RenderPanelWithLabel(label string, content string, width int, height int, active bool) string {
	var borderColor color.Color
	if active {
		borderColor = lipgloss.Color("86")
	} else {
		borderColor = lipgloss.Color("240")
	}
	border := lipgloss.RoundedBorder()
	topBorderStyler := lipgloss.NewStyle().Foreground(borderColor).Render
	topLeft := topBorderStyler(border.TopLeft)
	topRight := topBorderStyler(border.TopRight)
	labelStyle := lipgloss.NewStyle().Foreground(borderColor).Padding(0, 1)
	renderedLabel := labelStyle.Render(label)
	cellsShort := max(0, width-lipgloss.Width(topLeft+topRight+renderedLabel))
	gap := strings.Repeat(border.Top, cellsShort)
	top := topLeft + renderedLabel + topBorderStyler(gap) + topRight
	boxStyle := lipgloss.NewStyle().
		Border(border).
		BorderForeground(borderColor).
		BorderTop(false).
		Padding(1, 2).
		Width(width)
	if height > 0 {
		boxStyle = boxStyle.Height(height)
	}
	bottom := boxStyle.Render(content)
	return top + "\n" + bottom
}

// PadCell fits an already-styled string to an exact display width: it pads with
// trailing spaces when short and truncates (ANSI-aware, with an ellipsis) when
// long. Used to lock every list column to a fixed width so headers and rows align.
func PadCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	w := lipgloss.Width(s)
	switch {
	case w > width:
		return ansi.Truncate(s, width, "…")
	case w < width:
		return s + strings.Repeat(" ", width-w)
	default:
		return s
	}
}

func TruncateLongString(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max-1]) + "…"
	}
	return s
}

func Osc8(url, s string) string {
	return "\x1b]8;;" + url + "\x1b\\" + s + "\x1b]8;;\x1b\\"
}

const DateLayout = "2006-01-02"

// DateUnset is shown in place of a date the issue does not have.
const DateUnset = "—"

// DueSoonDays is how many days ahead of the due date the field starts warning.
const DueSoonDays = 3

type DueUrgency int

const (
	DueUnset DueUrgency = iota
	DueOverdue
	DueToday
	DueSoon
	DueLater
	DueClosed
)

func (u DueUrgency) String() string {
	switch u {
	case DueOverdue:
		return "overdue"
	case DueToday:
		return "today"
	case DueSoon:
		return "soon"
	case DueLater:
		return "later"
	case DueClosed:
		return "closed"
	default:
		return "unset"
	}
}

// FormatDate renders an ISO date for display, falling back to the raw value
// when it does not parse and to DateUnset when empty.
func FormatDate(iso string) string {
	if iso == "" {
		return DateUnset
	}
	d, err := time.Parse(DateLayout, iso)
	if err != nil {
		return iso
	}
	return d.Format("Jan 02")
}

// ClassifyDue buckets a due date by how much calendar time is left, counting
// whole days in now's location so "today" means today to the reader. A closed
// issue is never urgent: its deadline has stopped meaning anything.
func ClassifyDue(iso string, now time.Time, closed bool) DueUrgency {
	if iso == "" {
		return DueUnset
	}
	d, err := time.Parse(DateLayout, iso)
	if err != nil {
		return DueUnset
	}
	if closed {
		return DueClosed
	}

	loc := now.Location()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	due := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, loc)
	days := int(due.Sub(today).Hours() / 24)

	switch {
	case days < 0:
		return DueOverdue
	case days == 0:
		return DueToday
	case days <= DueSoonDays:
		return DueSoon
	default:
		return DueLater
	}
}

func dueStyle(u DueUrgency) (lipgloss.Style, string) {
	switch u {
	case DueOverdue:
		return DueOverdueStyle, IconDueOverdue
	case DueToday:
		return DueTodayStyle, IconDueToday
	case DueSoon:
		return DueSoonStyle, IconDueSoon
	case DueLater:
		return DueLaterStyle, IconDueLater
	case DueClosed:
		return DueClosedStyle, IconDueClosed
	default:
		return DueUnsetStyle, ""
	}
}

// RenderDue colors a due date by urgency and prefixes an icon signalling how
// close it is.
func RenderDue(iso string, now time.Time, closed bool) string {
	urgency := ClassifyDue(iso, now, closed)
	style, icon := dueStyle(urgency)

	text := FormatDate(iso)
	if icon != "" {
		text = icon + " " + text
	}
	return style.Render(text)
}
