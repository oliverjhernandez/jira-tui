// Package ui
package ui

import "github.com/charmbracelet/lipgloss"

var (
	PrimaryColor   = lipgloss.Color("15")
	secondaryColor = lipgloss.Color("240")
	// accentColor    = lipgloss.Color("42")

	BaseListPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(PrimaryColor).
				Padding(1, 2)

	BaseDetailPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(secondaryColor).
				Padding(1, 2)

	SeparatorStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderTop(true).
			Width(80) // TODO: this should be dynamic

	// selectedItemStyle = lipgloss.NewStyle().
	// 			Foreground(accentColor).
	// 			Bold(true)

	StatusBarStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	DetailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("63"))

	DetailFieldStyle = lipgloss.NewStyle().
				Width(30)

	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("63")).
				Bold(true)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	// Field Styles
	KeyFieldStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true).
			Width(12).
			Align(lipgloss.Left)

	StatusFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(12).
				Align(lipgloss.Left)

	SummaryFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(40).
				Align(lipgloss.Left)

	AssigneeFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(20).
				Align(lipgloss.Left)

	PriorityFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(10).
				Align(lipgloss.Left)

	StatusInProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("42")).
				Padding(0, 1).
				Bold(true)

	StatusDoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("42")).
			Padding(0, 1).
			Bold(true)

	StatusToDoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("153")).
			Padding(0, 1).
			Bold(true)

	StatusDefaultStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("240")).
				Padding(0, 1).
				Bold(true)
)
