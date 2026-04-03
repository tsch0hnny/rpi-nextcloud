package style

import "github.com/charmbracelet/lipgloss"

// Nextcloud-inspired color palette
var (
	ColorPrimary   = lipgloss.Color("#0082c9")
	ColorSecondary = lipgloss.Color("#00639a")
	ColorAccent    = lipgloss.Color("#00b4d8")
	ColorSuccess   = lipgloss.Color("#2ea043")
	ColorWarning   = lipgloss.Color("#d29922")
	ColorError     = lipgloss.Color("#f85149")
	ColorSurface   = lipgloss.Color("#1a1a2e")
	ColorBorder    = lipgloss.Color("#30363d")
	ColorText      = lipgloss.Color("#e6edf3")
	ColorSubtle    = lipgloss.Color("#8b949e")
	ColorDim       = lipgloss.Color("#484f58")
	ColorWhite     = lipgloss.Color("#ffffff")
)

// Reusable styles
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWhite).
			Background(ColorPrimary).
			Padding(0, 2)

	SubtitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorPrimary)

	DescriptionStyle = lipgloss.NewStyle().
				Foreground(ColorSubtle)

	TextStyle = lipgloss.NewStyle().
			Foreground(ColorText)

	BoldStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorText)

	SuccessStyle = lipgloss.NewStyle().
			Foreground(ColorSuccess)

	WarningStyle = lipgloss.NewStyle().
			Foreground(ColorWarning)

	ErrorStyle = lipgloss.NewStyle().
			Foreground(ColorError)

	CodeBlockStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Background(lipgloss.Color("#0d1117")).
			Padding(0, 1)

	BoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder).
			Padding(1, 2)

	ActiveBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(1, 2)

	ProgressBarFilled = lipgloss.NewStyle().
				Foreground(ColorPrimary).
				Bold(true)

	ProgressBarEmpty = lipgloss.NewStyle().
				Foreground(ColorDim)

	KeyHintStyle = lipgloss.NewStyle().
			Foreground(ColorSubtle)

	KeyStyle = lipgloss.NewStyle().
			Foreground(ColorAccent).
			Bold(true)

	StepNumberStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary).
			Bold(true)

	OptionalBadge = lipgloss.NewStyle().
			Foreground(ColorWarning).
			Bold(true).
			Padding(0, 1)

	ButtonActive = lipgloss.NewStyle().
			Foreground(ColorWhite).
			Background(ColorPrimary).
			Padding(0, 3).
			Bold(true)

	ButtonInactive = lipgloss.NewStyle().
			Foreground(ColorSubtle).
			Background(lipgloss.Color("#21262d")).
			Padding(0, 3)

	InputLabelStyle = lipgloss.NewStyle().
			Foreground(ColorText).
			Bold(true)

	SpinnerStyle = lipgloss.NewStyle().
			Foreground(ColorPrimary)

	DividerStyle = lipgloss.NewStyle().
			Foreground(ColorBorder)
)

// Divider builds a horizontal divider of the given width.
func Divider(width int) string {
	line := ""
	for i := 0; i < width; i++ {
		line += "─"
	}
	return DividerStyle.Render(line)
}
