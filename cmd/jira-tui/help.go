package main

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

const (
	helpModalWScale = 0.6
	helpModalHScale = 0.8
)

var helpKeyStyle = lipgloss.NewStyle().Bold(true).Foreground(ui.ThemeFg)

type helpBind struct {
	keys string
	desc string
}

type helpGroup struct {
	title string
	binds []helpBind
}

// helpGroups is the single source for the `?` help screen. Keep it in sync with
// the key handlers in main.go / list.go / detail.go / search.go when bindings
// change (there is no key registry — this is maintained by hand).
var helpGroups = []helpGroup{
	{"Global / Tabs", []helpBind{
		{"gt / gT", "Next / previous tab"},
		{"x", "Close current tab"},
		{"b", "Open epic board"},
		{"B", "Open saved-board picker"},
		{"P", "Open project picker"},
		{"v", "Toggle status / epic view"},
		{"?", "Toggle this help"},
		{"q / ctrl+c", "Quit"},
	}},
	{"Navigation", []helpBind{
		{"j / k", "Down / up"},
		{"gg / G", "Top / bottom"},
		{"ctrl+d / ctrl+u", "Half page down / up"},
		{"ctrl+f / ctrl+b", "Full page down / up"},
	}},
	{"List", []helpBind{
		{"enter", "Open issue"},
		{"alt+enter", "Open parent issue"},
		{"n", "New issue"},
		{"t", "Transition"},
		{"a", "Assign"},
		{"p", "Priority"},
		{"/", "Filter list"},
		{"ctrl+s", "Search issues"},
		{"ctrl+r", "Refresh"},
		{"y k / y K / y s", "Yank key / URL / summary"},
	}},
	{"Detail", []helpBind{
		{"tab / shift+tab", "Next / previous section"},
		{"[ / ]", "Previous / next section"},
		{"e", "Edit summary / description / comment / worklog"},
		{"E", "Set estimate"},
		{"t", "Transition"},
		{"a", "Assign"},
		{"p", "Priority"},
		{"c", "New comment"},
		{"d", "Delete comment / worklog"},
		{"w", "Log work"},
		{"l", "Link issue"},
		{"n", "New sub-task (sub-tasks section)"},
		{"gp", "Go to parent"},
		{"yy", "Yank focused text"},
		{"ctrl+r", "Refresh"},
		{"esc", "Back"},
	}},
	{"Search", []helpBind{
		{"ctrl+p / ctrl+n", "Previous / next result"},
		{"[ / ]", "Previous / next result"},
		{"enter", "Run query / open result"},
		{"esc", "Close"},
	}},
	{"Modals / forms", []helpBind{
		{"enter", "Confirm / submit"},
		{"esc", "Cancel"},
		{"tab / shift+tab", "Next / previous field"},
	}},
}

func buildHelpContent() string {
	maxKeys := 0
	for _, g := range helpGroups {
		for _, b := range g.binds {
			if len(b.keys) > maxKeys {
				maxKeys = len(b.keys)
			}
		}
	}

	var sb strings.Builder
	for gi, g := range helpGroups {
		if gi > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(ui.SectionTitleStyle.Render(g.title) + "\n")
		for _, b := range g.binds {
			keys := helpKeyStyle.Render(fmt.Sprintf("%-*s", maxKeys, b.keys))
			sb.WriteString("  " + keys + "   " + b.desc + "\n")
		}
	}
	return sb.String()
}

func (m *model) refreshHelpViewport() {
	w := ui.GetModalWidth(m.windowWidth, helpModalWScale) - ui.PanelOverheadWidth
	h := ui.GetModalHeight(m.windowHeight, helpModalHScale) - ui.PanelOverheadHeight - 2
	if w < 20 {
		w = 20
	}
	if h < 3 {
		h = 3
	}
	m.helpViewport.SetWidth(w)
	m.helpViewport.SetHeight(h)
	m.helpViewport.SetContent(buildHelpContent())
}

func (m model) openHelp() (tea.Model, tea.Cmd) {
	m.previousMode = m.mode
	m.mode = helpView
	m.helpViewport.SetYOffset(0)
	m.refreshHelpViewport()
	return m, nil
}

func (m model) updateHelpView(msg tea.Msg) (tea.Model, tea.Cmd) {
	if kp, ok := msg.(tea.KeyPressMsg); ok {
		switch kp.String() {
		case "esc", "q", "?":
			m.mode = m.previousMode
			m.lastKey = ""
			return m, nil
		case "j", "down":
			m.helpViewport.ScrollDown(1)
		case "k", "up":
			m.helpViewport.ScrollUp(1)
		case "ctrl+d":
			m.helpViewport.HalfPageDown()
		case "ctrl+u":
			m.helpViewport.HalfPageUp()
		case "ctrl+f":
			m.helpViewport.PageDown()
		case "ctrl+b":
			m.helpViewport.PageUp()
		case "G":
			m.helpViewport.GotoBottom()
		case "g":
			if m.lastKey == "g" {
				m.helpViewport.GotoTop()
				m.lastKey = ""
			} else {
				m.lastKey = "g"
			}
		}
	}
	return m, nil
}

func (m model) renderHelpView() string {
	footer := ui.StatusBarInfoStyle.Render("  j/k scroll · ctrl+d/u · esc close")
	return m.renderModal("Keymaps", m.helpViewport.View()+"\n"+footer, helpModalWScale, helpModalHScale)
}
