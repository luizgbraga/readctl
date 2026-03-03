package tui

import (
	"database/sql"
	"fmt"

	"github.com/charmbracelet/lipgloss"
	tea "charm.land/bubbletea/v2"
	"github.com/luizgbraga/readctl/internal/config"
	"github.com/luizgbraga/readctl/internal/storage"
	"github.com/luizgbraga/readctl/internal/tui/components"
	"github.com/luizgbraga/readctl/internal/tui/views"
)

type viewState int

const (
	viewBooks viewState = iota
	viewTopics
	viewChat
	viewDocument
)

type Model struct {
	state        viewState
	width        int
	height       int
	db           *sql.DB
	config       *config.Config
	showHelp     bool
	err          error
	helpModel    components.HelpModel
	modal        components.ModalModel
	booksView    views.BooksModel
	topicsView   views.TopicsModel
	chatView     views.ChatModel
	documentView views.DocumentModel
	currentBook  storage.Book
	currentTopic storage.Topic
}

func New(db *sql.DB, cfg *config.Config) Model {
	return Model{
		state:     viewBooks,
		db:        db,
		config:    cfg,
		helpModel: components.NewHelpModel(),
		modal:     components.NewModalModel(),
		booksView: views.NewBooksModel(db),
	}
}

func (m Model) Init() tea.Cmd {
	return m.booksView.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	// Handle view-specific messages first
	switch msg := msg.(type) {
	case views.SelectedBookMsg:
		// Store current book and switch to topics view
		m.currentBook = msg.Book
		m.topicsView = views.NewTopicsModel(m.db, msg.Book.ID, msg.Book.Title)
		// Set the size on the new topics view (it won't receive WindowSizeMsg)
		if m.width > 0 && m.height > 0 {
			m.topicsView.SetSize(m.width, m.height)
		}
		m.state = viewTopics
		// Update last accessed time
		if err := storage.UpdateLastAccessed(m.db, msg.Book.ID); err != nil {
			m.err = err
		}
		return m, m.topicsView.Init()

	case views.BackToBooksMsg:
		// Switch back to books view and reload
		m.state = viewBooks
		return m, m.booksView.Init()

	case views.BackToTopicsMsg:
		// Switch back to topics view
		m.state = viewTopics
		return m, m.topicsView.Init()

	case views.SelectedTopicMsg:
		// Switch to chat view for this topic
		m.currentTopic = msg.Topic
		m.chatView = views.NewChatModel(m.db, m.config, m.currentBook, msg.Topic)
		// Set the size on the new chat view
		if m.width > 0 && m.height > 0 {
			sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
			m.chatView, _ = m.chatView.Update(sizeMsg)
		}
		m.state = viewChat
		return m, m.chatView.Init()

	case views.ViewDocumentMsg:
		// Switch to document view
		m.currentTopic = msg.Topic
		m.documentView = views.NewDocumentModel(msg.Topic)
		if m.width > 0 && m.height > 0 {
			sizeMsg := tea.WindowSizeMsg{Width: m.width, Height: m.height}
			m.documentView, _ = m.documentView.Update(sizeMsg)
		}
		m.state = viewDocument
		return m, m.documentView.Init()
	}

	// Filter out terminal escape sequences globally (OSC responses, CSI sequences)
	// These come from terminal color queries and should never reach any view
	if msg, ok := msg.(tea.KeyMsg); ok {
		keyStr := msg.String()
		if isTerminalEscapeSequence(keyStr) {
			return m, nil
		}
	}

	// Handle window resize - propagate to all views
	if msg, ok := msg.(tea.WindowSizeMsg); ok {
		m.width = msg.Width
		m.height = msg.Height
		m.booksView, cmd = m.booksView.Update(msg)
		if m.state == viewTopics {
			m.topicsView, cmd = m.topicsView.Update(msg)
		}
		if m.state == viewChat {
			m.chatView, cmd = m.chatView.Update(msg)
		}
		return m, cmd
	}

	// Handle help overlay
	if m.showHelp {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "h", "esc":
				m.showHelp = false
				return m, nil
			case "q", "ctrl+c":
				return m, tea.Quit
			}
		}
		return m, nil
	}

	// Handle modal
	if m.modal.IsVisible() {
		if msg, ok := msg.(tea.KeyMsg); ok {
			switch msg.String() {
			case "y":
				// Confirm deletion
				m.modal.Hide()
				switch m.state {
				case viewBooks:
					cmd = m.booksView.DeleteSelected()
				case viewTopics:
					cmd = m.topicsView.DeleteSelected()
				}
				return m, cmd
			case "n", "esc":
				// Cancel deletion
				m.modal.Hide()
				// Reset deleting state in views
				if m.state == viewBooks {
					m.booksView.CancelDelete()
				} else if m.state == viewTopics {
					m.topicsView.CancelDelete()
				}
				return m, nil
			}
		}
		return m, nil
	}

	// Handle global keys (but not when a view is in input mode)
	if msg, ok := msg.(tea.KeyMsg); ok {
		// Check if any view is in creating mode (text input active)
		isInputActive := m.booksView.IsCreating() ||
			(m.state == viewTopics && m.topicsView.IsCreating()) ||
			m.state == viewChat // Chat view always has input active

		switch msg.String() {
		case "ctrl+c":
			// Always allow ctrl+c to quit
			return m, tea.Quit
		case "q":
			// Only quit if not in input mode
			if !isInputActive {
				return m, tea.Quit
			}
		case "h":
			// Only toggle help if not in input mode
			if !isInputActive {
				m.showHelp = !m.showHelp
				return m, nil
			}
		}
	}

	// Delegate to current view
	switch m.state {
	case viewBooks:
		m.booksView, cmd = m.booksView.Update(msg)
		// Show modal if entering delete mode
		if m.booksView.IsDeleting() {
			if book := m.booksView.SelectedBook(); book != nil {
				m.modal.Show(
					"Delete Book",
					fmt.Sprintf("Delete '%s' and all its topics?", book.Title),
				)
			}
		}
		return m, cmd

	case viewTopics:
		m.topicsView, cmd = m.topicsView.Update(msg)
		// Show modal if entering delete mode
		if m.topicsView.IsDeleting() {
			if topic := m.topicsView.SelectedTopic(); topic != nil {
				m.modal.Show(
					"Delete Topic",
					fmt.Sprintf("Delete topic '%s'?", topic.Name),
				)
			}
		}
		return m, cmd

	case viewChat:
		m.chatView, cmd = m.chatView.Update(msg)
		return m, cmd

	case viewDocument:
		m.documentView, cmd = m.documentView.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m Model) View() tea.View {
	if m.width == 0 {
		v := tea.NewView("Initializing...")
		v.AltScreen = true
		return v
	}

	// Render help overlay if visible
	if m.showHelp {
		var keys []components.KeyBinding
		switch m.state {
		case viewBooks:
			keys = components.BooksKeys()
		case viewTopics:
			keys = components.TopicsKeys()
		case viewChat:
			keys = components.ChatKeys()
		}
		m.helpModel.SetKeys(keys)
		v := tea.NewView(m.helpModel.View(m.width, m.height))
		v.AltScreen = true
		return v
	}

	// Render modal if visible
	if m.modal.IsVisible() {
		overlay := m.modal.View(m.width, m.height)
		v := tea.NewView(overlay)
		v.AltScreen = true
		return v
	}

	// Compose header + content + footer
	header := components.RenderHeader(m.getLocation(), m.width)

	// Use context-aware footer for chat view
	var footer string
	if m.state == viewChat {
		// Get mode from chat view (it tracks the current mode)
		footer = components.RenderFooterWithContextAndMode(m.getFooterHints(), m.chatView.GetContextPercent(), m.chatView.GetMode(), m.width)
	} else {
		footer = components.RenderFooter(m.getFooterHints(), m.width)
	}

	// Render content - chat view manages its own height via WindowSizeMsg
	content := m.renderCurrentView()

	// For non-chat views, constrain height to leave room for footer
	// Chat view handles its own layout including input area
	if m.state != viewChat {
		contentHeight := m.height - 2 // header + footer
		if contentHeight > 0 {
			content = lipgloss.NewStyle().
				Height(contentHeight).
				Render(content)
		}
	}

	view := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		content,
		footer,
	)

	v := tea.NewView(view)
	v.AltScreen = true
	v.MouseMode = tea.MouseModeCellMotion // Enable mouse scroll
	return v
}

func (m Model) getLocation() string {
	switch m.state {
	case viewBooks:
		return "Books"
	case viewTopics:
		return fmt.Sprintf("Books > %s", m.currentBook.Title)
	case viewChat:
		return fmt.Sprintf("Books > %s > %s", m.currentBook.Title, m.currentTopic.Name)
	case viewDocument:
		return fmt.Sprintf("Books > %s > %s (document)", m.currentBook.Title, m.currentTopic.Name)
	}
	return ""
}

func (m Model) getFooterHints() string {
	switch m.state {
	case viewBooks:
		return "n: new book • enter: open • d: delete • h: help • q: quit"
	case viewTopics:
		return "n: new topic • enter: resume • y: document • d: delete • esc: back • h: help • q: quit"
	case viewChat:
		if m.chatView.IsStreaming() {
			return "esc: cancel • waiting for response..."
		}
		hints := "enter: send • shift+enter: new line • esc: back"
		if m.chatView.HasSummary() {
			hints = "enter: send • shift+enter: new line • a: archive • esc: back"
		}
		return hints
	case viewDocument:
		return "esc: back to topics"
	}
	return ""
}

func (m Model) renderCurrentView() string {
	switch m.state {
	case viewBooks:
		return m.booksView.View()
	case viewTopics:
		return m.topicsView.View()
	case viewChat:
		return m.chatView.View()
	case viewDocument:
		return m.documentView.View()
	}
	return ""
}

// isTerminalEscapeSequence detects terminal escape sequences that should be filtered out
// These come from terminal color queries (OSC 10/11), cursor position reports (CSI R), etc.
func isTerminalEscapeSequence(s string) bool {
	if len(s) == 0 {
		return false
	}

	// Check for escape character at start or anywhere in string
	for _, r := range s {
		if r == '\x1b' || r == '\x9b' {
			return true
		}
	}

	// Check for OSC response patterns (starts with ])
	if s[0] == ']' {
		return true
	}

	// Check for common escape sequence content
	if len(s) > 5 {
		// OSC color responses contain "rgb:" or ";"
		for i := 0; i < len(s)-3; i++ {
			if s[i:i+4] == "rgb:" {
				return true
			}
		}
		// CSI cursor position report ends with "R" and contains ";"
		if s[len(s)-1] == 'R' {
			for _, r := range s {
				if r == ';' {
					return true
				}
			}
		}
	}

	return false
}
