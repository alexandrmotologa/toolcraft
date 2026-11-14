package tui

import "github.com/charmbracelet/lipgloss"

var (
	ColorPrimary   = lipgloss.Color("#7D56F4")
	ColorSecondary = lipgloss.Color("#04B575")
	ColorWarning   = lipgloss.Color("#FFA500")
	ColorDanger    = lipgloss.Color("#FF4D4D")
	ColorDim       = lipgloss.Color("#666666")
	ColorBg        = lipgloss.Color("#1A1A24")
	ColorFg        = lipgloss.Color("#EEEEEE")

	HeaderStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(ColorPrimary).
			Padding(0, 1).
			MarginBottom(1)

	FooterStyle = lipgloss.NewStyle().
			Foreground(ColorDim).
			MarginTop(1)

	ListPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorPrimary).
			Padding(0, 1)

	DetailPaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorDim).
			Padding(0, 1)

	ActivePaneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorSecondary).
			Padding(0, 1)

	SelectedItemStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#3A2D5C"))

	MethodGetStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#61AFFE")).
			Padding(0, 1)

	MethodPostStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#49CC90")).
			Padding(0, 1)

	MethodPutStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#000000")).
			Background(lipgloss.Color("#FCA130")).
			Padding(0, 1)

	MethodDeleteStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#F93E3E")).
				Padding(0, 1)

	BadgeSuccess = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorSecondary)

	BadgeWarning = lipgloss.NewStyle().
			Bold(true).
			Foreground(ColorWarning)
)
