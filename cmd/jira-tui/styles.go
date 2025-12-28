package main

import "charm.land/lipgloss/v2"

var (
	primaryColor   = lipgloss.Color("15")
	secondaryColor = lipgloss.Color("240")
	// accentColor    = lipgloss.Color("42")

	baseListPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(primaryColor).
				Padding(1, 2)

	detailPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(secondaryColor).
				Padding(1, 2).
				Height(20).
				Width(100)

	// selectedItemStyle = lipgloss.NewStyle().
	// 			Foreground(accentColor).
	// 			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(secondaryColor).
			Italic(true)

	detailHeaderStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("63"))

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Bold(true)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252"))

	// Field Styles
	keyFieldStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252")).
			Bold(true).
			Width(12).
			Align(lipgloss.Left)

	statusFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(12).
				Align(lipgloss.Left)

	summaryFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(40).
				Align(lipgloss.Left)

	assigneeFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(20).
				Align(lipgloss.Left)

	priorityFieldStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("252")).
				Bold(true).
				Width(10).
				Align(lipgloss.Left)

	statusInProgressStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("42")).
				Padding(0, 1).
				Bold(true)

	statusDoneStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("42")).
			Padding(0, 1).
			Bold(true)

	statusToDoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("153")).
			Padding(0, 1).
			Bold(true)

	statusDefaultStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("0")).
				Background(lipgloss.Color("240")).
				Padding(0, 1).
				Bold(true)
)
