package main

import (
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

type SavedBoardFormData struct {
	SelectedIndex int
	Form          *huh.Form
}

func NewSavedBoardFormData() *SavedBoardFormData {
	options := make([]huh.Option[int], len(savedBoards))
	for i, b := range savedBoards {
		options[i] = huh.NewOption(b.Title, i)
	}

	d := &SavedBoardFormData{SelectedIndex: 0}
	d.Form = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[int]().
				Title("Open Board").
				Options(options...).
				Value(&d.SelectedIndex),
		),
	).WithWidth(50)

	return d
}

func (m model) openSavedBoardPicker() (tea.Model, tea.Cmd) {
	m.previousMode = m.mode
	m.savedBoardData = NewSavedBoardFormData()
	m.mode = savedBoardPickerView
	return m, m.savedBoardData.Form.Init()
}

func (m model) updateSavedBoardPickerView(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if keyPressMsg, ok := msg.(tea.KeyPressMsg); ok {
		switch keyPressMsg.String() {
		case "esc":
			m.mode = m.previousMode
			m.savedBoardData = nil
			return m, nil
		}
	}

	form, cmd := m.savedBoardData.Form.Update(msg)
	if f, ok := form.(*huh.Form); ok {
		m.savedBoardData.Form = f
		cmds = append(cmds, cmd)
	}

	if m.savedBoardData.Form.State == huh.StateCompleted {
		idx := m.savedBoardData.SelectedIndex
		m.savedBoardData = nil
		if idx >= 0 && idx < len(savedBoards) {
			b := savedBoards[idx]
			return m.openBoardTab(b.Title, b.JQL, tabSavedBoard)
		}
		m.mode = m.previousMode
	}

	return m, tea.Batch(cmds...)
}

func (m model) renderSavedBoardPickerView() string {
	var content string
	if m.savedBoardData != nil {
		content = m.savedBoardData.Form.View()
	}
	return m.renderModal("Open Board", content, 0.3, 0.3)
}
