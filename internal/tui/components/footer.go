package components

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/luizgbraga/readctl/internal/tui/styles"
)
func RenderFooter(hints string, width int) string {
	// Left-align the hints
	style := styles.Footer.Width(width)
	return style.Render(hints)
}
func RenderFooterWithContext(hints string, contextPercent int, width int) string {
	// Cap percentage at 100% for display
	displayPercent := contextPercent
	if displayPercent > 100 {
		displayPercent = 100
	}

	// Determine color based on thresholds
	var color lipgloss.Color
	switch {
	case contextPercent >= 95:
		color = lipgloss.Color("#ff5555") // red
	case contextPercent >= 80:
		color = lipgloss.Color("#ffd700") // yellow
	default:
		color = lipgloss.Color("#6c6c6c") // gray
	}

	// Format context indicator with color
	contextStyle := lipgloss.NewStyle().Foreground(color)
	contextText := contextStyle.Render(fmt.Sprintf("%d%%", displayPercent))
	fullText := fmt.Sprintf("%s | Context: %s", hints, contextText)

	// Apply footer style
	style := styles.Footer.Width(width)
	return style.Render(fullText)
}
func RenderFooterWithContextAndMode(hints string, contextPercent int, mode string, width int) string {
	// Cap percentage at 100% for display
	displayPercent := contextPercent
	if displayPercent > 100 {
		displayPercent = 100
	}

	// Determine context color based on thresholds
	var contextColor lipgloss.Color
	switch {
	case contextPercent >= 95:
		contextColor = lipgloss.Color("#ff5555") // red
	case contextPercent >= 80:
		contextColor = lipgloss.Color("#ffd700") // yellow
	default:
		contextColor = lipgloss.Color("#6c6c6c") // gray
	}

	// Format mode label - uppercase with green styling
	modeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true)
	modeText := modeStyle.Render(fmt.Sprintf("[%s]", strings.ToUpper(mode)))

	// Format context indicator with color
	contextStyle := lipgloss.NewStyle().Foreground(contextColor)
	contextText := contextStyle.Render(fmt.Sprintf("%d%%", displayPercent))

	// Compose: hints | mode | context
	fullText := fmt.Sprintf("%s | %s | Context: %s", hints, modeText, contextText)

	// Apply footer style
	style := styles.Footer.Width(width)
	return style.Render(fullText)
}
