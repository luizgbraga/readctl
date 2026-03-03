package views

import (
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/luizgbraga/readctl/internal/storage"
	"github.com/luizgbraga/readctl/internal/tui/styles"
)


type DocumentModel struct {
	topic    storage.Topic
	content  string
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

// documentLoadedMsg is sent when document loads from storage
type documentLoadedMsg struct {
	content string
}


func NewDocumentModel(topic storage.Topic) DocumentModel {
	return DocumentModel{
		topic: topic,
	}
}


func (m DocumentModel) Init() tea.Cmd {
	return m.loadDocumentCmd()
}


func (m DocumentModel) Update(msg tea.Msg) (DocumentModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		if !m.ready {
			m.viewport = viewport.New(
				viewport.WithWidth(msg.Width-4),
				viewport.WithHeight(msg.Height-4),
			)
			m.ready = true
			m.viewport.SetContent(m.renderDocument())
		} else {
			m.viewport = viewport.New(
				viewport.WithWidth(msg.Width-4),
				viewport.WithHeight(msg.Height-4),
			)
			m.viewport.SetContent(m.renderDocument())
		}

		return m, nil

	case documentLoadedMsg:
		m.content = msg.content
		if m.ready {
			m.viewport.SetContent(m.renderDocument())
		}
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			// Return to topics view
			return m, func() tea.Msg {
				return BackToTopicsMsg{}
			}

		case "up", "down", "pgup", "pgdown", "home", "end":
			// Delegate scrolling to viewport
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

	case tea.MouseMsg:
		// Pass mouse events to viewport for scroll wheel
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}


func (m DocumentModel) View() string {
	if !m.ready {
		return "Loading document..."
	}

	// Render viewport wrapped in padding
	viewportContent := lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.width - 4).
		Render(m.viewport.View())

	return viewportContent
}

// renderDocument converts document content to rendered markdown
func (m DocumentModel) renderDocument() string {
	if m.content == "" {
		// Empty state
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.SubtleColor).
			MarginTop(2).
			Align(lipgloss.Center)
		return emptyStyle.Render("No document yet. Use /doc in chat to generate one.")
	}

	// Calculate width for glamour
	glamourWidth := m.width - 12
	if glamourWidth < 40 {
		glamourWidth = 40
	}

	// Render markdown using glamour
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(glamourWidth),
	)
	if err != nil {
		return m.content // Fallback to plain text
	}

	rendered, err := r.Render(m.content)
	if err != nil {
		return m.content // Fallback to plain text
	}

	return strings.TrimSpace(rendered)
}

// loadDocumentCmd loads document from storage
func (m DocumentModel) loadDocumentCmd() tea.Cmd {
	return func() tea.Msg {
		content, err := storage.LoadDocument(m.topic.ChatFileUUID)
		if err != nil {
			// Return empty content on error
			return documentLoadedMsg{content: ""}
		}
		return documentLoadedMsg{content: content}
	}
}
