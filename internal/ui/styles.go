// Package ui
package ui

import (
	"github.com/charmbracelet/lipgloss"
)

const (
	ColWidthType     = 4
	ColWidthKey      = 12
	ColWidthSummary  = 50
	ColWidthStatus   = 13
	ColWidthAssignee = 20
	ColWidthPriority = 1
	ColWidthCursor   = 2
	ColWidthEmpty    = 2
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
	ThemeBorder       = CatSurface1
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
	ThemePriorityHighest = CatRed
	ThemePriorityHigh    = CatPeach
	ThemePriorityMedium  = CatYellow
	ThemePriorityLow     = CatGreen
	ThemePriorityLowest  = CatTeal

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
	IconSubtask       = `󰘑`
	IconImprovement   = ""
	IconDefault       = `󰧞`

	// Priority
	IconPriorityHighest = `󰶼`
	IconPriorityHigh    = `󰄿`
	IconPriorityMedium  = `󰇼`
	IconPriorityLow     = `󰄼`
	IconPriorityLowest  = `󰶹`

	// Status indicators
	IconStatusInProgress = `󰐊`
	IconStatusDone       = `󰗠`
	IconStatusValidation = `󱀝`
	IconStatusToDo       = `󱃔`
	IconStatusBlocked    = `󰜺`
	IconStatusDefault    = ``

	// UI elements
	IconCursor     = "▌"
	IconExpanded   = ""
	IconCollapsed  = ""
	IconComment    = ""
	IconAttachment = ""
	IconTime       = ""
)

// ============================================================================
// PANEL STYLES
// ============================================================================

var (
	// Focused panel - bright border
	PanelStyleActive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ThemeBorderActive).
				Padding(1, 2)

	// Unfocused panel - dim border
	PanelStyleInactive = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ThemeBorder).
				Padding(1, 2)

	// Panel title
	PanelTitleStyle = lipgloss.NewStyle().
			Foreground(ThemeAccent).
			Bold(true).
			Padding(0, 1)

	PanelTitleInactiveStyle = lipgloss.NewStyle().
				Foreground(ThemeFgDim).
				Bold(true).
				Padding(0, 1)
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
// FIELD STYLES (for list columns)
// ============================================================================

var (
	KeyFieldStyle = lipgloss.NewStyle().
			Foreground(ThemeKey).
			Width(ColWidthKey).
			Align(lipgloss.Left)

	SummaryFieldStyle = lipgloss.NewStyle().
				Foreground(ThemeFg).
				Width(ColWidthSummary).
				Align(lipgloss.Left)

	AssigneeFieldStyle = lipgloss.NewStyle().
				Foreground(ThemeFgMuted).
				Width(ColWidthAssignee).
				Align(lipgloss.Left)

	PriorityFieldStyle = lipgloss.NewStyle().
				Width(ColWidthPriority).
				Align(lipgloss.Left)

	StatusFieldStyle = lipgloss.NewStyle().
				Width(ColWidthStatus).
				MarginRight(1).
				Align(lipgloss.Left)
)

// ============================================================================
// STATUS BADGE STYLES
// ============================================================================

var (
	StatusInProgressStyle = lipgloss.NewStyle().
				Foreground(ThemeStatusInProgress).
				Width(ColWidthStatus).
				Bold(true)

	StatusDoneStyle = lipgloss.NewStyle().
			Foreground(ThemeStatusDone).
			Width(ColWidthStatus).
			Bold(true)

	StatusValidationStyle = lipgloss.NewStyle().
				Foreground(ThemeStatusValidation).
				Width(ColWidthStatus).
				Bold(true)

	StatusToDoStyle = lipgloss.NewStyle().
			Foreground(ThemeStatusToDo).
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
	PriorityBaseStyle    = lipgloss.NewStyle()
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
	TypeSubTaskStyle = TypeBaseStyle.Foreground(ThemeFgMuted)
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
				Bold(true).
				Width(10)

	DetailValueStyle = lipgloss.NewStyle().
				Foreground(ThemeFg)

	SeparatorStyle = lipgloss.NewStyle().
			Foreground(ThemeBorder)

	SectionTitleStyle = lipgloss.NewStyle().
				Foreground(ThemeFgMuted).
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
// STATUS BAR STYLES
// ============================================================================

var (
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(ThemeFgDim).
			Italic(true)

	StatusBarKeyStyle = lipgloss.NewStyle().
				Foreground(ThemeAccent).
				Bold(true)

	StatusBarDescStyle = lipgloss.NewStyle().
				Foreground(ThemeFgDim)
)

var (
	EmptyHeaderSpace = lipgloss.NewStyle().Width(ColWidthEmpty).Render("")
	TypeHeader       = lipgloss.NewStyle().Width(ColWidthType).MarginLeft(ColWidthCursor).Render("TYPE")
	KeyHeader        = lipgloss.NewStyle().Width(ColWidthKey).Render("KEY")
	PriorityHeader   = lipgloss.NewStyle().Width(ColWidthPriority).Render("")
	SummaryHeader    = lipgloss.NewStyle().Width(ColWidthSummary).Render("SUMMARY")
	StatusHeader     = lipgloss.NewStyle().Width(ColWidthStatus - ColWidthEmpty).Render("STATUS")
	AssigneeHeader   = lipgloss.NewStyle().Width(ColWidthAssignee).Render("ASSIGNEE")
)

// ============================================================================
// INFO PANEL STYLES
// ============================================================================

var (
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
