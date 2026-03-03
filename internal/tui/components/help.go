package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/luizgbraga/readctl/internal/tui/styles"
)

type HelpModel struct {
	keys []KeyBinding
}

type KeyBinding struct {
	Key  string
	Desc string
}
func NewHelpModel() HelpModel {
	return HelpModel{}
}
func (h *HelpModel) SetKeys(keys []KeyBinding) {
	h.keys = keys
}
func (h HelpModel) View(width, height int) string {
	var lines []string
	lines = append(lines, styles.HelpKeyStyle.Render("Keyboard Shortcuts"))
	lines = append(lines, "")

	for _, kb := range h.keys {
		key := styles.HelpKeyStyle.Render(kb.Key)
		desc := styles.HelpDescStyle.Render(kb.Desc)
		lines = append(lines, key+" - "+desc)
	}

	lines = append(lines, "")
	lines = append(lines, styles.HelpDescStyle.Render("Press h to close"))

	content := strings.Join(lines, "\n")
	helpBox := styles.HelpStyle.Render(content)

	// Center the help overlay on screen
	return lipgloss.Place(
		width,
		height,
		lipgloss.Center,
		lipgloss.Center,
		helpBox,
	)
}
func BooksKeys() []KeyBinding {
	return []KeyBinding{
		{"n", "new book"},
		{"enter", "open book"},
		{"d", "delete book"},
		{"h", "toggle help"},
		{"q", "quit"},
	}
}
func TopicsKeys() []KeyBinding {
	return []KeyBinding{
		{"n", "new topic"},
		{"enter", "resume topic"},
		{"d", "delete topic"},
		{"esc", "back to books"},
		{"h", "toggle help"},
		{"q", "quit"},
	}
}
func ChatKeys() []KeyBinding {
	return []KeyBinding{
		{"enter", "send message"},
		{"shift+enter", "new line"},
		{"r", "retry (on error)"},
		{"esc", "back to topics"},
		{"h", "toggle help"},
		{"q", "quit"},
	}
}
