package main

import (
	"bytes"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"github.com/muesli/ansi"
	"github.com/muesli/reflow/truncate"
	"github.com/muesli/termenv"
	"github.com/oliverjhernandez/jira-tui/internal/jira"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type whitespace struct {
	style termenv.Style
	chars string
}

func filterIssues(issues []jira.Issue, filter string) []jira.Issue {
	var filtered []jira.Issue

	for _, i := range issues {
		if issueMatchesFilter(i, filter) {
			filtered = append(filtered, i)
		}
	}

	return filtered
}

func renderStatusBadge(status string) string {
	if strings.ToLower(status) == "selected for development" {
		status = "To Do"
	}

	statusLower := strings.ToLower(status)

	if strings.Contains(statusLower, "trabajando") {
		return ui.StatusInProgressStyle.Render(status)
	} else if strings.Contains(statusLower, "done") {
		return ui.StatusDoneStyle.Render(status)
	} else if strings.Contains(statusLower, "backlog") || strings.Contains(statusLower, "to do") {
		return ui.StatusToDoStyle.Render(status)
	}

	return ui.StatusDefaultStyle.Render(status)
}

func truncateLongString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

func renderField(label, value string) string {
	return ui.DetailFieldStyle.Render(
		ui.DetailLabelStyle.Render(label+": "),
		ui.DetailValueStyle.Render(value),
	)
}

func issueMatchesFilter(issue jira.Issue, filter string) bool {
	filterLower := strings.ToLower(filter)
	return strings.Contains(strings.ToLower(issue.Summary), filterLower) ||
		strings.Contains(strings.ToLower(issue.Key), filterLower) ||
		strings.Contains(strings.ToLower(issue.Status), filterLower)
}

func PlaceOverlay(
	x, y int,
	fg, bg string,
	shadow bool, opts ...WhitespaceOption,
) string {
	fgLines, fgWidth := getLines(fg)
	bgLines, bgWidth := getLines(bg)
	bgHeight := len(bgLines)
	fgHeight := len(fgLines)

	if shadow {
		shadowbg := ""
		shadowchar := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#333333")).
			Render("░")
		for i := 0; i <= fgHeight; i++ {
			if i == 0 {
				shadowbg += " " + strings.Repeat(" ", fgWidth) + "\n"
			} else {
				shadowbg += " " + strings.Repeat(shadowchar, fgWidth) + "\n"
			}
		}

		fg = PlaceOverlay(0, 0, fg, shadowbg, false, opts...)
		fgLines, fgWidth = getLines(fg)
		fgHeight = len(fgLines)
	}

	if fgWidth >= bgWidth && fgHeight >= bgHeight {
		// FIXME: return fg or bg?
		return fg
	}
	// TODO: allow placement outside of the bg box?
	x = clamp(x, 0, bgWidth-fgWidth)
	y = clamp(y, 0, bgHeight-fgHeight)

	ws := &whitespace{}
	for _, opt := range opts {
		opt(ws)
	}

	var b strings.Builder
	for i, bgLine := range bgLines {
		if i > 0 {
			b.WriteByte('\n')
		}
		if i < y || i >= y+fgHeight {
			b.WriteString(bgLine)
			continue
		}

		pos := 0
		if x > 0 {
			left := truncate.String(bgLine, uint(x))
			pos = ansi.PrintableRuneWidth(left)
			b.WriteString(left)
			if pos < x {
				b.WriteString(ws.render(x - pos))
				pos = x
			}
		}

		fgLine := fgLines[i-y]
		b.WriteString(fgLine)
		pos += ansi.PrintableRuneWidth(fgLine)

		right := cutLeft(bgLine, pos)
		bgWidth := ansi.PrintableRuneWidth(bgLine)
		rightWidth := ansi.PrintableRuneWidth(right)
		if rightWidth <= bgWidth-pos {
			b.WriteString(ws.render(bgWidth - rightWidth - pos))
		}

		b.WriteString(right)
	}

	return b.String()
}

func getLines(s string) (lines []string, widest int) {
	lines = strings.Split(s, "\n")

	for _, l := range lines {
		w := ansi.PrintableRuneWidth(l)
		if widest < w {
			widest = w
		}
	}

	return lines, widest
}

func cutLeft(s string, cutWidth int) string {
	var (
		pos    int
		isAnsi bool
		ab     bytes.Buffer
		b      bytes.Buffer
	)
	for _, c := range s {
		var w int
		if c == ansi.Marker || isAnsi {
			isAnsi = true
			ab.WriteRune(c)
			if ansi.IsTerminator(c) {
				isAnsi = false
				if bytes.HasSuffix(ab.Bytes(), []byte("[0m")) {
					ab.Reset()
				}
			}
		} else {
			w = runewidth.RuneWidth(c)
		}

		if pos >= cutWidth {
			if b.Len() == 0 {
				if ab.Len() > 0 {
					b.Write(ab.Bytes())
				}
				if pos-cutWidth > 1 {
					b.WriteByte(' ')
					continue
				}
			}
			b.WriteRune(c)
		}
		pos += w
	}
	return b.String()
}

func clamp(v, lower, upper int) int {
	return min(max(v, lower), upper)
}

func (w whitespace) render(width int) string {
	if w.chars == "" {
		w.chars = " "
	}

	r := []rune(w.chars)
	j := 0
	b := strings.Builder{}

	// Cycle through runes and print them into the whitespace.
	for i := 0; i < width; {
		b.WriteRune(r[j])
		j++
		if j >= len(r) {
			j = 0
		}
		i += ansi.PrintableRuneWidth(string(r[j]))
	}

	// Fill any extra gaps white spaces. This might be necessary if any runes
	// are more than one cell wide, which could leave a one-rune gap.
	short := width - ansi.PrintableRuneWidth(b.String())
	if short > 0 {
		b.WriteString(strings.Repeat(" ", short))
	}

	return w.style.Styled(b.String())
}

// WhitespaceOption sets a styling rule for rendering whitespace.
type WhitespaceOption func(*whitespace)
