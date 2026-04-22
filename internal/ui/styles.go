// Package ui
package ui

import (
	"charm.land/lipgloss/v2"
)

const (
	ColWidthType      = 4
	ColWidthKey       = 12
	ColWidthSummary   = 24
	ColWidthStatus    = 18
	ColWidthAssignee  = 20
	ColWidthReporter  = 20
	ColWidthPriority  = 1
	ColWidthCursor    = 2
	ColWidthEmpty     = 2
	ColWidthTimeSpent = 8

	// Total width of a list row (cursor + all columns + spacing)
	ListRowWidth = ColWidthCursor + ColWidthType + ColWidthEmpty + ColWidthKey + ColWidthPriority + ColWidthEmpty + ColWidthSummary + ColWidthEmpty + ColWidthReporter + ColWidthEmpty + ColWidthStatus + ColWidthEmpty + ColWidthAssignee + ColWidthEmpty + ColWidthTimeSpent
)

var (
	// Backgrounds
	ThemeBg          = CatBase
	ThemeBgDark      = CatCrust
	ThemeBgLight     = CatSurface0
	ThemeBgHighlight = CatSurface1

	// Foregrounds
	ThemeFg      = CatText
	ThemeFgMuted = CatSubtext0
	ThemeFgDim   = CatOverlay1

	// Borders
	ThemeBorder       = CatOverlay2
	ThemeBorderActive = CatBlue

	// Accents
	ThemeAccent    = CatBlue
	ThemeAccentAlt = CatMauve

	// Semantic colors
	ThemeSuccess = CatGreen
	ThemeWarning = CatYellow
	ThemeError   = CatRed
	ThemeInfo    = CatSapphire

	// Status colors
	ThemeStatusInProgress = CatGreen
	ThemeStatusDone       = CatGreen
	ThemeStatusValidation = CatLavender
	ThemeStatusToDo       = CatYellow
	ThemeStatusBlocked    = CatRed
	ThemeStatusDefault    = CatOverlay1

	// Priority colors
	ThemePriorityCritical = CatRed
	ThemePriorityHighest  = CatRed
	ThemePriorityHigh     = CatPeach
	ThemePriorityMedium   = CatYellow
	ThemePriorityLow      = CatGreen
	ThemePriorityLowest   = CatTeal

	// Special
	ThemeKey     = CatMauve
	ThemeComment = CatOverlay0
	ThemeMention = CatSapphire
	ThemeLink    = CatBlue
)

// ============================================================================
// ICONS
// ============================================================================

var (
	// Issue types
	IconBug           = `󰃤`
	IconTask          = `󰄬`
	IconStory         = `󰂺`
	IconEpic          = `󱐋`
	IconInvestigacion = `󰍉`
	IconSubTask       = `󰘑`
	IconImprovement   = ""
	IconDefault       = `󰧞`

	// Priority
	IconPriorityCritical = `󰈸`
	IconPriorityHighest  = `󰶼`
	IconPriorityHigh     = `󰄿`
	IconPriorityMedium   = `󰇼`
	IconPriorityLow      = `󰄼`
	IconPriorityLowest   = `󰶹`

	// Status indicators
	IconStatusInProgress = `󰐊`
	IconStatusDone       = `󰗠`
	IconStatusReady      = `󰑣`
	IconStatusValidation = `󱀝`
	IconStatusToDo       = `󱃔`
	IconStatusBacklog    = ``
	IconStatusBlocked    = `󰜺`
	IconStatusSelected   = ``

	// UI elements
	IconCursor     = "▌"
	IconExpanded   = ""
	IconCollapsed  = ""
	IconComment    = ""
	IconAttachment = ""
	IconTime       = ""

	// Error
	IconError = ""
)

// ============================================================================
// PANEL STYLES
// ============================================================================

var (
	// Focused panel - bright border
	PanelActiveSecondaryStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(ThemeBorderActive).
					Padding(1, 2)

	PanelActiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("86")).
				Padding(1, 2)

	PanelInactiveStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("240")).
				Padding(1, 2)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))
)

// ============================================================================
// MODAL STYLES
// ============================================================================

var (
	ModalTextInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	ModalBlockInputStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	Modal3InputFormStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("62"))

	ModalMultiSelectFormStyle = lipgloss.NewStyle().
					Border(lipgloss.RoundedBorder()).
					BorderForeground(lipgloss.Color("62"))
)

// ============================================================================
// LIST ITEM STYLES
// ============================================================================

var (
	ColumnHeaderStyle = lipgloss.NewStyle().
				Foreground(ThemeFgDim).
				Bold(true)

	SelectedRowStyle = lipgloss.NewStyle().
				Background(ThemeBgHighlight).
				Foreground(ThemeFg).
				Bold(true)

	NormalRowStyle = lipgloss.NewStyle().
			Foreground(ThemeFg)

	CursorStyle = lipgloss.NewStyle().
			Foreground(ThemeAccent).
			Background(ThemeAccent).
			Bold(true)

	CursorBarStyle = lipgloss.NewStyle().
			Foreground(ThemeAccent).
			Bold(true)
)

// ============================================================================
// FIELD STYLES
// ============================================================================

var (
	KeyFieldStyle = lipgloss.NewStyle().
			Foreground(ThemeKey).
			Align(lipgloss.Left)
		// Width(ColWidthKey).

	SummaryFieldStyle = lipgloss.NewStyle().
				Foreground(ThemeFg).
				Width(ColWidthSummary).
				Align(lipgloss.Left)

	AssigneeFieldStyle = lipgloss.NewStyle().
				Foreground(ThemeFgMuted).
				Width(ColWidthAssignee).
				Align(lipgloss.Left)

	ReporterFieldStyle = lipgloss.NewStyle().
				Foreground(ThemeFgMuted).
				Width(ColWidthReporter).
				Align(lipgloss.Left)

	PriorityFieldStyle = lipgloss.NewStyle().
				Width(ColWidthPriority).
				Align(lipgloss.Left)

	StatusFieldStyle = lipgloss.NewStyle().
				Width(ColWidthStatus).
				MarginRight(1).
				Align(lipgloss.Left)

	TimeSpentFieldStyle = lipgloss.NewStyle().
				Foreground(ThemeFgDim).
				Width(ColWidthTimeSpent).
				Align(lipgloss.Right)
)

// ============================================================================
// STATUS BADGE STYLES
// ============================================================================

var (
	StatusInProgressStyle = lipgloss.NewStyle().
				Foreground(CatGreen).
				Width(ColWidthStatus).
				Bold(true)

	StatusDoneStyle = lipgloss.NewStyle().
			Foreground(CatOverlay1).
			Width(ColWidthStatus).
			Bold(true)

	StatusReadyStyle = lipgloss.NewStyle().
				Foreground(CatOverlay2).
				Width(ColWidthStatus).
				Bold(true)

	StatusValidationStyle = lipgloss.NewStyle().
				Foreground(CatPeach).
				Width(ColWidthStatus).
				Bold(true)

	StatusToDoStyle = lipgloss.NewStyle().
			Foreground(ThemeStatusToDo).
			Width(ColWidthStatus).
			Bold(true)

	StatusBacklogStyle = lipgloss.NewStyle().
				Foreground(CatOverlay1).
				Width(ColWidthStatus).
				Bold(true)

	StatusSelectedStyle = lipgloss.NewStyle().
				Foreground(CatLavender).
				Width(ColWidthStatus).
				Bold(true)

	StatusBlockedStyle = lipgloss.NewStyle().
				Foreground(ThemeStatusBlocked).
				Width(ColWidthStatus).
				Bold(true)

	StatusDefaultStyle = lipgloss.NewStyle().
				Foreground(ThemeBorder).
				Width(ColWidthStatus)
)

// ============================================================================
// PRIORITY STYLES
// ============================================================================

var (
	PriorityBaseStyle     = lipgloss.NewStyle()
	PriorityCriticalStyle = PriorityBaseStyle.
				Foreground(ThemePriorityCritical).
				Bold(true)
	PriorityHighestStyle = PriorityBaseStyle.
				Foreground(ThemePriorityHighest).
				Bold(true)

	PriorityHighStyle = PriorityBaseStyle.
				Foreground(ThemePriorityHigh).
				Bold(true)

	PriorityMediumStyle = PriorityBaseStyle.
				Foreground(ThemePriorityMedium)

	PriorityLowStyle = PriorityBaseStyle.
				Foreground(ThemePriorityLow)

	PriorityLowestStyle = PriorityBaseStyle.
				Foreground(ThemePriorityLowest)
)

// ============================================================================
// TYPE STYLES
// ============================================================================

var (
	TypeBaseStyle    = lipgloss.NewStyle()
	TypeBugStyle     = TypeBaseStyle.Foreground(ThemeError)
	TypeTaskStyle    = TypeBaseStyle.Foreground(ThemeInfo)
	TypeStoryStyle   = TypeBaseStyle.Foreground(ThemeSuccess)
	TypeEpicStyle    = TypeBaseStyle.Foreground(ThemeAccentAlt)
	TypeInvestStyle  = TypeBaseStyle.Foreground(ThemeAccentAlt)
	TypeSubtaskStyle = TypeBaseStyle.Foreground(ThemeFgMuted)
)

// ============================================================================
// DETAIL VIEW STYLES
// ============================================================================

var (
	DetailHeaderStyle = lipgloss.NewStyle().
				Foreground(ThemeAccent).
				Bold(true)

	DetailLabelStyle = lipgloss.NewStyle().
				Foreground(ThemeAccent).
				Bold(true)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(ThemeFg)

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(ThemeBorder)

	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(ThemeFgMuted).
				PaddingLeft(4).
				Bold(true)
)

// ============================================================================
// COMMENT STYLES
// ============================================================================

var (
	CommentAuthorStyle = lipgloss.NewStyle().
				Foreground(ThemeAccentAlt).
				Bold(true)

	CommentTimestampStyle = lipgloss.NewStyle().
				Foreground(ThemeFgDim).
				Italic(true)

	CommentBodyStyle = lipgloss.NewStyle().
				Foreground(ThemeFg)

	MentionStyle = lipgloss.NewStyle().
			Foreground(ThemeMention).
			Background(ThemeBgLight).
			Padding(0, 1)
)

// ============================================================================
// WORKLOGS STYLES
// ============================================================================

var (
	WorklogsAuthorStyle = lipgloss.NewStyle().
				Foreground(ThemeAccentAlt).
				Bold(true)

	WorklogsTimestampStyle = lipgloss.NewStyle().
				Foreground(ThemeFgDim).
				Italic(true)

	WorkLogsDescriptionStyle = lipgloss.NewStyle().
					Foreground(ThemeFgDim)
)

// ============================================================================
// STATUS BAR STYLES
// ============================================================================

var (
	StatusBarInfoStyle = lipgloss.NewStyle().
				Foreground(CatSubtext1).
				Italic(true)

	StatusBarLoadingStyle = lipgloss.NewStyle().
				Foreground(CatOverlay1).
				Italic(true)

	StatusBarErrorStyle = lipgloss.NewStyle().
				Foreground(CatRed).
				Bold(true)
)

var (
	EmptyHeaderSpace = lipgloss.NewStyle().Width(ColWidthEmpty).Render("")
	TypeHeader       = lipgloss.NewStyle().Width(ColWidthType).MarginLeft(ColWidthCursor).Render("TYPE")
	KeyHeader        = lipgloss.NewStyle().Width(ColWidthKey).Render("KEY")
	PriorityHeader   = lipgloss.NewStyle().Width(ColWidthPriority).Render("")
	SummaryHeader    = lipgloss.NewStyle().Width(ColWidthSummary).Render("SUMMARY")
	ReporterHeader   = lipgloss.NewStyle().Width(ColWidthReporter).Render("REPORTER")
	StatusHeader     = lipgloss.NewStyle().Width(ColWidthStatus - ColWidthEmpty).Render("STATUS")
	AssigneeHeader   = lipgloss.NewStyle().Width(ColWidthAssignee).Render("ASSIGNEE")
)

// ============================================================================
// INFO PANEL STYLES
// ============================================================================

var (
	DimTextStyle = lipgloss.NewStyle().
			Foreground(ThemeFgDim).
			Italic(true)

	InfoPanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ThemeBorder).
			Padding(0, 2)

	InfoPanelUserStyle = lipgloss.NewStyle().
				Foreground(ThemeAccent).
				Bold(true)

	InfoPanelProjectStyle = lipgloss.NewStyle().
				Foreground(ThemeFgMuted)

	InfoPanelProjectSepStyle = lipgloss.NewStyle().
					Foreground(ThemeBorder)

	InfoPanelCountLabelStyle = lipgloss.NewStyle().
					Foreground(ThemeFgMuted)

	InfoPanelTotalStyle = lipgloss.NewStyle().
				Foreground(ThemeFgDim).
				Italic(true)

	// Status count icons
	IconInfoInProgress = lipgloss.NewStyle().Foreground(ThemeStatusInProgress).Render("●")
	IconInfoToDo       = lipgloss.NewStyle().Foreground(ThemeStatusToDo).Render("○")
	IconInfoDone       = lipgloss.NewStyle().Foreground(ThemeStatusDone).Render("✓")
)

// ============================================================================
// MARKDOWN STYLES
// ============================================================================

var (
	BoldStyle       = lipgloss.NewStyle().Bold(true)
	ItalicStyle     = lipgloss.NewStyle().Italic(true)
	InlineCodeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Background(lipgloss.Color("236"))

	HeadingStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("39"))

	CodeBlockStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("236")).
			Padding(0, 1).
			MarginTop(1).
			MarginBottom(1)
)
