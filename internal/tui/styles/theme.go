package styles

import "github.com/charmbracelet/lipgloss"

var (
	// Dark theme colors
	darkBg     = lipgloss.Color("#1e1e1e")
	darkFg     = lipgloss.Color("#e0e0e0")
	accentCyan = lipgloss.Color("#04B575")
	mutedGray  = lipgloss.Color("#6c6c6c")
	errorRed   = lipgloss.Color("#ff5555")

	// Base style for the app
	AppStyle = lipgloss.NewStyle().
		Padding(0)

	// Header style
	Header = lipgloss.NewStyle().
		Foreground(darkFg).
		Background(darkBg).
		Bold(true).
		Padding(0, 1)

	// Footer style
	Footer = lipgloss.NewStyle().
		Foreground(mutedGray).
		Background(darkBg).
		Padding(0, 1)

	// Selected item accent
	SelectedItem = lipgloss.NewStyle().
		Foreground(accentCyan).
		Bold(true)

	// Error text style
	ErrorText = lipgloss.NewStyle().
		Foreground(errorRed).
		Bold(true)

	// Help overlay styles
	HelpStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentCyan).
		Padding(1, 2).
		Background(darkBg).
		Foreground(darkFg)

	HelpKeyStyle = lipgloss.NewStyle().
		Foreground(accentCyan).
		Bold(true)

	HelpDescStyle = lipgloss.NewStyle().
		Foreground(darkFg)

	// Modal styles
	ModalStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentCyan).
		Padding(1, 2).
		Background(darkBg).
		Foreground(darkFg)

	ModalTitleStyle = lipgloss.NewStyle().
		Foreground(accentCyan).
		Bold(true)

	// List styles
	ListTitleStyle = lipgloss.NewStyle().
		Foreground(accentCyan).
		Bold(true)

	ListSelectedTitleStyle = lipgloss.NewStyle().
		Foreground(accentCyan).
		Bold(true)

	ListSelectedDescStyle = lipgloss.NewStyle().
		Foreground(darkFg)

	ListDescStyle = lipgloss.NewStyle().
		Foreground(mutedGray)

	// Subtle color for empty states
	SubtleColor = mutedGray
)
