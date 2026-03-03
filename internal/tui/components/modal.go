package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/luizgbraga/readctl/internal/tui/styles"
)

type ModalModel struct {
	visible    bool
	title      string
	message    string
	confirmKey string
	cancelKey  string
}
func NewModalModel() ModalModel {
	return ModalModel{
		confirmKey: "y",
		cancelKey:  "n",
	}
}
func (m *ModalModel) Show(title, message string) {
	m.visible = true
	m.title = title
	m.message = message
}
func (m *ModalModel) Hide() {
	m.visible = false
}
func (m ModalModel) IsVisible() bool {
	return m.visible
}
func (m ModalModel) View(width, height int) string {
	if !m.visible {
		return ""
	}

	var lines []string
	lines = append(lines, styles.ModalTitleStyle.Render(m.title))
	lines = append(lines, "")
	lines = append(lines, styles.HelpDescStyle.Render(m.message))
	lines = append(lines, "")
	prompt := styles.HelpDescStyle.Render("(" + m.confirmKey + "/" + m.cancelKey + ")")
	lines = append(lines, prompt)

	content := strings.Join(lines, "\n")
	modalBox := styles.ModalStyle.Render(content)

	// Center the modal on screen
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		modalBox,
	)
}
