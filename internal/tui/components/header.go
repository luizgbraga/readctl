package components

import (
	"github.com/luizgbraga/readctl/internal/tui/styles"
)
func RenderHeader(location string, width int) string {
	// Center the location text
	style := styles.Header.Width(width)
	return style.Render(location)
}
