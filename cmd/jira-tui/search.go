package main

import (
	"log"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/oliverjhernandez/jira-tui/internal/ui"
)

type SearchFormData struct {
	Query string
	Form  *huh.Form
	Err   error
}

func NewSearchFormData() *SearchFormData {
	e := &SearchFormData{
		Query: "",
	}
	e.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Search Issue").
				Placeholder("DEV-123").
				Value(&e.Query),
		),
	).WithTheme(huh.ThemeCatppuccin()).WithWidth(40)

	return e
}

func (m model) updateSearchView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "esc":
			m.mode = listView
			m.searchData = nil
			return m, nil
		}
	}

	form, cmd := m.searchData.Form.Update(msg)

	if f, ok := form.(*huh.Form); ok {
		m.searchData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.searchData.Form.State == huh.StateCompleted {
		m.searchData.Err = nil
		m.loadingDetail = true
		log.Printf("Search form completed with value: %s", m.searchData.Query)
		cmds = append(cmds, m.fetchIssueDetail(m.searchData.Query))
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderSearchView() string {
	log.Printf("=== renderSearchView called ===")

	bg := m.renderListView()

	var modalContent strings.Builder

	modalContent.WriteString(m.searchData.Form.View())
	modalContent.WriteString("\n")

	if m.searchData.Err != nil {
		modalContent.WriteString(ui.ErrorStyle.Render("Error: " + m.searchData.Err.Error()))
	}

	// footer := strings.Join([]string{
	// 	ui.RenderKeyBind("enter", "submit"),
	// 	ui.RenderKeyBind("esc", "cancel"),
	// }, "  ")
	// modalContent.WriteString(footer)

	contentWidth := ui.GetPanelWidth(m.windowWidth) - 6
	contentHeight := 6
	x, y := ui.GetCenteredModalPosition(m.windowWidth, m.windowHeight, contentWidth, contentHeight)

	styledModal := ui.ModalTextInputStyle.Render(modalContent.String())
	overlay := PlaceOverlay(x, y, styledModal, bg, false)

	return overlay
}
