package styles

import "github.com/charmbracelet/lipgloss"

// Text Styles
var (
	// HeaderStyle is used for section headers and titles
	HeaderStyle = lipgloss.NewStyle().
			Foreground(Purple).
			Bold(true).
			MarginBottom(1)

	// SubheaderStyle is for subsection headers
	SubheaderStyle = lipgloss.NewStyle().
			Foreground(Cyan).
			Bold(true)

	// LabelStyle is for input labels and field names
	LabelStyle = lipgloss.NewStyle().
			Foreground(Gray).
			MarginRight(1)

	// ValueStyle is for displaying values
	ValueStyle = lipgloss.NewStyle().
			Foreground(White).
			Bold(true)

	// DimStyle is for secondary or muted text
	DimStyle = lipgloss.NewStyle().
			Foreground(Gray)

	// SuccessTextStyle is for success messages
	SuccessTextStyle = lipgloss.NewStyle().
				Foreground(Green).
				Bold(true)

	// SuccessBgStyle is for success counts/badges with background
	SuccessBgStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(Green).
			Padding(0, 1).
			Bold(true)

	// WarningTextStyle is for warning messages
	WarningTextStyle = lipgloss.NewStyle().
				Foreground(Yellow).
				Bold(true)

	// ErrorTextStyle is for error messages
	ErrorTextStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true)

	// ErrorBgStyle is for error counts/badges with background
	ErrorBgStyle = lipgloss.NewStyle().
			Foreground(White).
			Background(Red).
			Padding(0, 1).
			Bold(true)

	// DocStyle is the default document style with minimal margins
	DocStyle = lipgloss.NewStyle().Margin(0, 1).Align(lipgloss.Left)

	// HighlightStyle is for command highlights (e.g. /run)
	HighlightStyle = lipgloss.NewStyle().
			Foreground(Purple).
			Bold(true)
)

// Box Styles
var (
	// BoxStyle is the default box with purple border
	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Purple).
			Padding(1, 2)

	// InfoBoxStyle is for informational boxes with cyan border
	InfoBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Cyan).
			Padding(1, 2)

	// SuccessBoxStyle is for success messages with green border
	SuccessBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Green).
			Padding(1, 2)

	// ErrorBoxStyle is for error messages with red border
	ErrorBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Red).
			Padding(1, 2)

	// WarningBoxStyle is for warnings with yellow border
	WarningBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Yellow).
			Padding(1, 2)
)

// Progress and Status Styles
var (
	// ProgressBarStyle is for progress indicators
	ProgressBarStyle = lipgloss.NewStyle().
				Foreground(Purple)

	// ProgressCompleteStyle is for completed progress
	ProgressCompleteStyle = lipgloss.NewStyle().
				Foreground(Green).
				Bold(true)

	// ProgressFailedStyle is for failed progress
	ProgressFailedStyle = lipgloss.NewStyle().
				Foreground(Red).
				Bold(true)

	// SpinnerStyle is for loading spinners
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Purple)
)

// Badge Styles
var (
	// BadgePassStyle is for pass badges
	BadgePassStyle = lipgloss.NewStyle().
			Foreground(Green).
			Bold(true).
			Padding(0, 1)

	// BadgeFailStyle is for fail badges
	BadgeFailStyle = lipgloss.NewStyle().
			Foreground(Red).
			Bold(true).
			Padding(0, 1)

	// BadgeRunningStyle is for running/in-progress badges
	BadgeRunningStyle = lipgloss.NewStyle().
				Foreground(Yellow).
				Bold(true).
				Padding(0, 1)

	// BadgePendingStyle is for pending badges
	BadgePendingStyle = lipgloss.NewStyle().
				Foreground(Gray).
				Bold(true).
				Padding(0, 1)
)

// List Styles
var (
	// SelectedItemStyle is for selected items in lists
	SelectedItemStyle = lipgloss.NewStyle().
				Foreground(Purple).
				Bold(true).
				PaddingLeft(2)

	// UnselectedItemStyle is for non-selected items
	UnselectedItemStyle = lipgloss.NewStyle().
				Foreground(White).
				PaddingLeft(4)

	// DisabledItemStyle is for disabled items
	DisabledItemStyle = lipgloss.NewStyle().
				Foreground(Gray).
				PaddingLeft(4)
)

// Helper Functions

// RenderBadge creates a styled badge with text
func RenderBadge(text string, style lipgloss.Style) string {
	return style.Render(text)
}

// RenderPass returns a styled PASS badge
func RenderPass() string {
	return RenderBadge(" PASS ", BadgePassStyle)
}

// RenderFail returns a styled FAIL badge
func RenderFail() string {
	return RenderBadge(" FAIL ", BadgeFailStyle)
}

// RenderRunning returns a styled RUNNING badge
func RenderRunning() string {
	return RenderBadge(" RUN ", BadgeRunningStyle)
}

// RenderPending returns a styled PENDING badge
func RenderPending() string {
	return RenderBadge(" PENDING ", BadgePendingStyle)
}

// RenderError returns a styled error message
func RenderError(msg string) string {
	return ErrorBoxStyle.Render("✗ " + msg)
}

// RenderSuccess returns a styled success message
func RenderSuccess(msg string) string {
	return SuccessBoxStyle.Render("✓ " + msg)
}

// RenderWarning returns a styled warning message
func RenderWarning(msg string) string {
	return WarningBoxStyle.Render("⚠ " + msg)
}

// RenderInfo returns a styled info message
func RenderInfo(msg string) string {
	return InfoBoxStyle.Render("ℹ " + msg)
}
