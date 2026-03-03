package views

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/luizgbraga/readctl/internal/config"
	"github.com/luizgbraga/readctl/internal/llm"
	"github.com/luizgbraga/readctl/internal/research"
	"github.com/luizgbraga/readctl/internal/storage"
	"github.com/luizgbraga/readctl/internal/tui/styles"
)

// ChatModel manages the chat view for a topic
type ChatModel struct {
	db               *sql.DB
	config           *config.Config
	llmClient        *llm.Client
	book             storage.Book // Book context for system prompt
	topic            storage.Topic
	messages         []storage.Message
	viewport         viewport.Model
	input            textarea.Model
	streamingContent string // Accumulator for current assistant message
	displayedChars   int    // For typewriter effect
	isStreaming      bool
	streamCancel     context.CancelFunc // Cancel function for current stream
	streamCancelled  bool               // True if user cancelled the stream
	lastError        *llm.StreamErrorMsg
	retryCount       int
	contextPercent   int                  // Current context usage percentage (0-100)
	hasSummary       bool                 // True if conversation has been summarized
	archiveExpanded  bool                 // True if user toggled archive viewing
	archivedMsgs     []storage.Message    // Loaded archive messages (if expanded)
	documentExists              bool              // True if document exists for this topic
	awaitingRewriteInstructions bool              // True if waiting for /rewrite-doc instructions
	confirmingDocOverwrite      bool              // True if waiting for document overwrite confirmation
	generatingDocument          bool              // True if document generation is in progress
	showModePicker              bool              // True if mode picker modal is visible
	modePickerList              list.Model        // Bubbles list for mode selection
	showCommandHints            bool              // True if showing command autocomplete
	commandHints                []string          // Filtered command suggestions
	selectedHint                int               // Currently selected hint index
	streamProcessor  *llm.StreamProcessor          // Handles tool use during streaming
	isSearching      bool                          // True if executing tool searches
	searchQueries    []string                      // Search queries being executed
	width            int
	height           int
	ready            bool
}

// Available slash commands
var slashCommands = []string{"/mode", "/doc", "/rewrite-doc"}

// Message types for chat updates
type chatLoadedMsg struct {
	messages []storage.Message
}

type chatSavedMsg struct{}

type charRevealMsg struct{}

type BackToTopicsMsg struct{}

type summarizeStartMsg struct{}

type summarizeCompleteMsg struct {
	summary  string
	messages []storage.Message // New message list: [summary msg] + [recent msgs]
}

type summarizeErrorMsg struct {
	err error
}

type documentGeneratedMsg struct {
	content string
	uuid    string
}

type documentSavedMsg struct{}

type updateTopicModeMsg struct {
	mode string
}

// modeItem is a list item for mode selection
type modeItem struct {
	name string
}

func (i modeItem) FilterValue() string { return i.name }
func (i modeItem) Title() string       { return i.name }
func (i modeItem) Description() string { return "" }

// compactDelegate is a minimal list delegate for mode selection (1 line per item)
type compactDelegate struct{}

func (d compactDelegate) Height() int                               { return 1 }
func (d compactDelegate) Spacing() int                              { return 0 }
func (d compactDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd { return nil }
func (d compactDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	i, ok := item.(modeItem)
	if !ok {
		return
	}

	str := i.name
	fn := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888")).Render
	if index == m.Index() {
		fn = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true).Render
		str = "> " + str
	} else {
		str = "  " + str
	}

	fmt.Fprint(w, fn(str))
}

func NewChatModel(db *sql.DB, cfg *config.Config, book storage.Book, topic storage.Topic) ChatModel {
	// Create LLM client with optional research capabilities
	llmClient := llm.NewClient(cfg.AnthropicAPIKey, cfg.Model, cfg.FirecrawlAPIKey)

	// Create textarea for input
	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Focus()
	ta.ShowLineNumbers = false
	ta.CharLimit = 0 // No character limit
	ta.SetHeight(1)  // Start with single line

	// Create mode picker list with compact delegate (1 line per item)
	modes := []list.Item{
		modeItem{name: "Scholar"},
		modeItem{name: "Socratic"},
		modeItem{name: "Dialectical"},
		modeItem{name: "Provocateur"},
	}
	modeList := list.New(modes, compactDelegate{}, 20, 6) // Compact: 4 items + title + spacing
	modeList.Title = "Select Mode"
	modeList.SetShowStatusBar(false)
	modeList.SetFilteringEnabled(false)
	modeList.SetShowHelp(false)
	modeList.SetShowPagination(false)

	return ChatModel{
		db:             db,
		config:         cfg,
		llmClient:      llmClient,
		book:           book,
		topic:          topic,
		input:          ta,
		modePickerList: modeList,
	}
}


func (m ChatModel) Init() tea.Cmd {
	// Check if document exists
	m.documentExists = storage.DocumentExists(m.topic.ChatFileUUID)
	return m.loadChatCmd()
}


func (m ChatModel) Update(msg tea.Msg) (ChatModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate viewport height
		// Account for: app header (1) + app footer (1) + input label (1) + input (dynamic) + padding (2)
		appChrome := 2          // app header + app footer
		inputLabel := 1         // "Message:" label
		inputHeight := m.input.Height()
		padding := 3            // viewport padding + spacing
		verticalMargins := appChrome + inputLabel + inputHeight + padding

		viewportHeight := msg.Height - verticalMargins
		if viewportHeight < 5 {
			viewportHeight = 5 // minimum
		}

		if !m.ready {
			m.viewport = viewport.New(
				viewport.WithWidth(msg.Width-4),
				viewport.WithHeight(viewportHeight),
			)
			m.ready = true
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		} else {
			// Recreate viewport with new dimensions
			m.viewport = viewport.New(
				viewport.WithWidth(msg.Width-4),
				viewport.WithHeight(viewportHeight),
			)
			m.viewport.SetContent(m.renderMessages())
		}

		// Update input width
		m.input.SetWidth(msg.Width - 4)

		return m, nil

	case chatLoadedMsg:
		m.messages = msg.messages
		// Calculate initial context percentage
		m.contextPercent = m.calculateContextPercent()
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return m, nil

	case chatSavedMsg:
		// Chat saved successfully
		return m, nil

	case llm.StreamStartMsg:
		// Stream started - blur input and create processor
		m.input.Blur()
		m.streamProcessor = llm.NewStreamProcessor(msg)
		return m, m.streamProcessor.ProcessNext()

	case llm.StreamDeltaMsg:
		// Received incremental text - accumulate and update display
		m.streamingContent += msg.Text

		// Update viewport with streaming content
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}

		// Continue processing stream
		if m.streamProcessor != nil {
			return m, m.streamProcessor.ProcessNext()
		}
		return m, llm.ProcessStreamEvent(msg.Stream)

	case llm.StreamContinueMsg:
		// Continue processing stream (non-text event)
		if m.streamProcessor != nil {
			return m, m.streamProcessor.ProcessNext()
		}
		return m, llm.ProcessStreamEvent(msg.Stream)

	case llm.StreamToolUseMsg:
		// Claude wants to use research tools - execute them
		m.isSearching = true
		// Extract search queries from tool inputs
		m.searchQueries = nil
		for _, tc := range msg.ToolCalls {
			var input map[string]string
			if err := json.Unmarshal(tc.Input, &input); err == nil {
				if q, ok := input["query"]; ok {
					m.searchQueries = append(m.searchQueries, q)
				} else if t, ok := input["topic"]; ok {
					m.searchQueries = append(m.searchQueries, t)
				}
			}
		}
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return m, m.executeToolsCmd(msg)

	case llm.StreamCompleteMsg:
		// Stream completed - save assistant message
		content := msg.Content
		if content == "" {
			content = m.streamingContent
		}

		if content != "" {
			assistantMsg := storage.Message{
				Role:      "assistant",
				Content:   content,
				Timestamp: time.Now().Unix(),
			}
			m.messages = append(m.messages, assistantMsg)
		}

		m.isStreaming = false
		m.streamingContent = ""
		m.retryCount = 0
		m.lastError = nil

		// Recalculate context percentage after message
		m.contextPercent = m.calculateContextPercent()

		// Re-focus input after streaming completes
		m.input.Focus()

		// Save chat and update viewport
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}

		// Check if summarization is needed
		summarizeCmd := m.checkAndSummarizeIfNeeded()

		return m, tea.Batch(m.saveChatCmd(), summarizeCmd)

	case toolsExecutedMsg:
		// Tool results received - continue LLM response with results
		m.isSearching = false
		m.searchQueries = nil
		ctx, cancel := context.WithCancel(context.Background())
		m.streamCancel = cancel
		return m, m.llmClient.StreamResponseWithToolResults(
			ctx,
			m.messages,
			msg.convCtx,
			msg.mode,
			msg.toolCalls,
			msg.results,
		)

	case summarizeStartMsg:
		// Summarization triggered - show indicator and start summarization
		// Blur input to prevent typing during summarization
		m.input.Blur()
		return m, m.summarizeOldMessagesCmd()

	case summarizeCompleteMsg:
		// Summarization complete - update messages
		m.messages = msg.messages
		m.hasSummary = true

		// Recalculate context percentage
		m.contextPercent = m.calculateContextPercent()

		// Re-focus input
		m.input.Focus()

		// Update viewport
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}

		return m, m.saveChatCmd()

	case summarizeErrorMsg:
		// Summarization failed - log error but continue
		// In production, might show user-facing error
		m.input.Focus()
		return m, nil

	case documentGeneratedMsg:
		m.generatingDocument = false
		m.documentExists = true
		m.input.Focus()
		if err := storage.SaveDocument(msg.uuid, msg.content); err != nil {
			m.lastError = &llm.StreamErrorMsg{
				Message: fmt.Sprintf("Failed to save document: %v", err),
			}
		}
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return m, nil

	case updateTopicModeMsg:
		// Mode was updated successfully in DB
		// The topic.Mode field was already updated before sending the command
		return m, nil

	case llm.StreamErrorMsg:
		// Ignore errors if user cancelled the stream
		if m.streamCancelled {
			m.streamCancelled = false
			return m, nil
		}

		// Handle streaming error
		m.lastError = &msg
		m.isStreaming = false

		// Auto-retry once
		if m.retryCount < 1 && msg.ErrType == "rate_limit" {
			m.retryCount++
			// Wait for retry duration then retry
			return m, tea.Tick(msg.Retry, func(t time.Time) tea.Msg {
				return m.retryLastMessage()
			})
		}

		// Re-focus input so user can type
		m.input.Focus()

		// Show error in viewport
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
		}

		return m, nil

	case tea.KeyMsg:
		keyStr := msg.String()

		// If mode picker is visible, delegate to it
		if m.showModePicker {
			switch keyStr {
			case "enter":
				// Get selected mode
				selected := m.modePickerList.SelectedItem()
				if modeItem, ok := selected.(modeItem); ok {
					// Convert to lowercase for storage
					modeName := strings.ToLower(modeItem.name)
					// Update topic mode in DB
					m.showModePicker = false
					m.topic.Mode = modeName
					m.input.Focus()
					return m, m.updateTopicModeCmd(modeName)
				}
				return m, nil
			case "esc":
				// Already handled above
				m.showModePicker = false
				m.input.Focus()
				return m, nil
			default:
				// Delegate to list for navigation
				m.modePickerList, cmd = m.modePickerList.Update(msg)
				return m, cmd
			}
		}

		// Handle special keys only when input is empty
		inputEmpty := m.input.Value() == ""

		switch keyStr {
		case "ctrl+c":
			return m, tea.Quit

		case "esc":
			// If mode picker is visible, hide it
			if m.showModePicker {
				m.showModePicker = false
				m.input.Focus()
				return m, nil
			}

			// If streaming, cancel it
			if m.isStreaming {
				if m.streamCancel != nil {
					m.streamCancel()
					m.streamCancel = nil
				}
				m.isStreaming = false
				m.streamCancelled = true // Mark as user-cancelled to ignore error
				// Keep partial content if any was received
				if m.streamingContent != "" {
					assistantMsg := storage.Message{
						Role:      "assistant",
						Content:   m.streamingContent + "\n\n[cancelled]",
						Timestamp: time.Now().Unix(),
					}
					m.messages = append(m.messages, assistantMsg)
				}
				m.streamingContent = ""
				m.input.Focus()
				if m.ready {
					m.viewport.SetContent(m.renderMessages())
					m.viewport.GotoBottom()
				}
				return m, m.saveChatCmd()
			}
			// Return to topics view
			return m, func() tea.Msg {
				return BackToTopicsMsg{}
			}

		case "r":
			// Retry last message if there was an error and input is empty
			if m.lastError != nil && inputEmpty {
				m.lastError = nil
				return m, m.retryLastMessage()
			}

		case "a":
			// Toggle archive viewing if input is empty and not streaming
			if inputEmpty && !m.isStreaming && m.hasSummary {
				m.archiveExpanded = !m.archiveExpanded
				if m.archiveExpanded {
					// Load archive
					archived, err := storage.LoadChatArchive(m.topic.ChatFileUUID)
					if err == nil {
						m.archivedMsgs = archived
					}
				} else {
					// Clear archive
					m.archivedMsgs = nil
				}
				// Update viewport
				if m.ready {
					m.viewport.SetContent(m.renderMessages())
					m.viewport.GotoBottom()
				}
				return m, nil
			}

		case "tab":
			// Tab completes the selected command hint
			if m.showCommandHints && len(m.commandHints) > 0 {
				// Insert the selected command
				m.input.SetValue(m.commandHints[m.selectedHint])
				m.showCommandHints = false
				m.commandHints = nil
				return m, nil
			}

		case "up", "down", "pgup", "pgdown":
			// Navigate command hints if showing
			if m.showCommandHints && len(m.commandHints) > 0 {
				if keyStr == "up" {
					m.selectedHint--
					if m.selectedHint < 0 {
						m.selectedHint = len(m.commandHints) - 1
					}
				} else if keyStr == "down" {
					m.selectedHint++
					if m.selectedHint >= len(m.commandHints) {
						m.selectedHint = 0
					}
				}
				return m, nil
			}
			// Handle scrolling only when input is empty
			if inputEmpty {
				m.viewport, cmd = m.viewport.Update(msg)
				return m, cmd
			}

		case "enter":
			// Check if shift or alt is pressed for new line
			key := msg.Key()
			if key.Mod&tea.ModShift != 0 || key.Mod&tea.ModAlt != 0 {
				// Let textarea handle new line
				m.input, cmd = m.input.Update(msg)

				// Auto-expand: increase height based on line count
				lines := strings.Count(m.input.Value(), "\n") + 1
				newHeight := min(lines, 5) // Max 5 lines visible
				m.input.SetHeight(newHeight)

				return m, cmd
			}

			// Submit message
			content := m.input.Value()
			if content != "" {
				// Check for /mode command (exact match)
				if content == "/mode" {
					m.showModePicker = true
					m.input.Reset()
					m.input.SetHeight(1)
					if m.ready {
						m.viewport.SetContent(m.renderMessages())
						m.viewport.GotoBottom()
					}
					return m, nil
				}

				// Check for document commands
				if handled, cmd := m.handleDocumentCommand(content); handled {
					return m, cmd
				}
				// Create user message
				userMsg := storage.Message{
					Role:      "user",
					Content:   content,
					Timestamp: time.Now().Unix(),
				}
				m.messages = append(m.messages, userMsg)

				// Recalculate context percentage after user message
				m.contextPercent = m.calculateContextPercent()

				// Clear input
				m.input.Reset()
				m.input.SetHeight(1)

				// Update viewport
				if m.ready {
					m.viewport.SetContent(m.renderMessages())
					m.viewport.GotoBottom()
				}

				// Start streaming response with cancellable context
				m.isStreaming = true
				m.streamingContent = ""
				m.streamCancelled = false
				ctx, cancel := context.WithCancel(context.Background())
				m.streamCancel = cancel

				return m, tea.Batch(
					m.saveChatCmd(),
					m.llmClient.StreamResponse(ctx, m.messages, m.convContext(), m.topic.Mode),
				)
			}

			return m, nil
		}

		// Delegate to input for other keys (typing)
		m.input, cmd = m.input.Update(msg)

		// Update command hints based on input
		m.updateCommandHints()

		return m, cmd

	case tea.MouseMsg:
		// Pass mouse events to viewport for scroll wheel
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	// Pass other messages to viewport
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}


func (m ChatModel) View() string {
	if !m.ready {
		return "Loading chat..."
	}

	// Render viewport with messages
	viewportContent := lipgloss.NewStyle().
		Padding(1, 2).
		Width(m.width - 4).
		Render(m.viewport.View())

	// Render command hints if showing
	var commandHintsView string
	if m.showCommandHints && len(m.commandHints) > 0 {
		var hintParts []string
		for i, hint := range m.commandHints {
			style := lipgloss.NewStyle().Foreground(lipgloss.Color("#888888"))
			if i == m.selectedHint {
				style = style.Foreground(lipgloss.Color("#04B575")).Bold(true)
			}
			hintParts = append(hintParts, style.Render(hint))
		}
		commandHintsView = lipgloss.NewStyle().
			Padding(0, 2).
			Render(strings.Join(hintParts, "  "))
	}

	// Style input based on whether it's a valid command
	inputContent := m.input.Value()
	var inputLabel string
	if isExactCommand(inputContent) {
		inputLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#04B575")).Bold(true).Render("Command:")
	} else {
		inputLabel = styles.HelpKeyStyle.Render("Message:")
	}

	inputArea := lipgloss.NewStyle().
		Padding(0, 2).
		MarginBottom(1).
		Render(lipgloss.JoinVertical(
			lipgloss.Left,
			inputLabel,
			m.input.View(),
		))

	// Show error if present
	var errorMsg string
	if m.lastError != nil {
		errorMsg = lipgloss.NewStyle().
			Foreground(styles.ErrorText.GetForeground()).
			Padding(0, 2).
			Render(fmt.Sprintf("Error: %s (press 'r' to retry)", m.lastError.Message))
	}

	// Combine all elements
	parts := []string{viewportContent}
	if errorMsg != "" {
		parts = append(parts, errorMsg)
	}
	if commandHintsView != "" {
		parts = append(parts, commandHintsView)
	}
	parts = append(parts, inputArea)

	view := lipgloss.JoinVertical(lipgloss.Left, parts...)

	// If mode picker is visible, render it as an overlay
	if m.showModePicker {
		view = m.renderModePickerOverlay(view)
	}

	return view
}

// renderModePickerOverlay renders the mode picker modal as an overlay
func (m ChatModel) renderModePickerOverlay(baseView string) string {
	// Render the mode picker list
	listView := m.modePickerList.View()

	// Create modal box
	modalStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#04B575")).
		Padding(1, 2).
		Width(40)

	modalContent := modalStyle.Render(listView)

	// Add hint at bottom
	hintStyle := lipgloss.NewStyle().
		Foreground(styles.SubtleColor).
		Align(lipgloss.Center).
		Width(40)
	hint := hintStyle.Render("enter: select • esc: cancel")

	modal := lipgloss.JoinVertical(lipgloss.Center, modalContent, hint)

	// Center the modal over the base view
	overlay := lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#1a1a1a")),
	)

	return overlay
}

// renderMessages renders all messages in the chat
func (m ChatModel) renderMessages() string {
	if len(m.messages) == 0 && !m.isStreaming && !m.archiveExpanded {
		// Empty state
		emptyStyle := lipgloss.NewStyle().
			Foreground(styles.SubtleColor).
			MarginTop(2).
			Align(lipgloss.Center)
		return emptyStyle.Render(fmt.Sprintf("Start a conversation about %s", m.topic.Name))
	}

	var rendered []string

	// If archive is expanded and we have archived messages, show them first
	if m.archiveExpanded && len(m.archivedMsgs) > 0 {
		// Separator
		separatorStyle := lipgloss.NewStyle().
			Foreground(styles.SubtleColor).
			Padding(1, 2).
			Bold(true)
		rendered = append(rendered, separatorStyle.Render("--- Archived Messages ---"))

		// Render archived messages with dimmed style
		for i := range m.archivedMsgs {
			rendered = append(rendered, m.renderMessageBubble(m.archivedMsgs[i], false))
		}

		// Separator before current conversation
		rendered = append(rendered, separatorStyle.Render("--- Current Conversation ---"))
	}

	// Show last 20 messages
	startIdx := 0
	if len(m.messages) > 20 {
		startIdx = len(m.messages) - 20
	}

	for i := startIdx; i < len(m.messages); i++ {
		msg := m.messages[i]
		rendered = append(rendered, m.renderMessageBubble(msg, false))
	}

	// If streaming, show the streaming content
	if m.isStreaming && !m.isSearching {
		if m.streamingContent != "" {
			// Show streaming content as assistant message
			streamingMsg := storage.Message{
				Role:    "assistant",
				Content: m.streamingContent,
			}
			rendered = append(rendered, m.renderMessageBubble(streamingMsg, true))
		}
	}

	// Show document generation indicator
	if m.generatingDocument {
		thinkingStyle := lipgloss.NewStyle().
			Foreground(styles.SubtleColor).
			Padding(0, 2)
		rendered = append(rendered, thinkingStyle.Render("Generating document..."))
	}

	// Show search indicator with queries
	if m.isSearching {
		searchStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#04B575")).
			Padding(0, 2)
		rendered = append(rendered, searchStyle.Render("🔍 Researching..."))

		// Show each query being searched
		queryStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			Padding(0, 4)
		for _, q := range m.searchQueries {
			rendered = append(rendered, queryStyle.Render("→ "+q))
		}
	}

	return lipgloss.JoinVertical(lipgloss.Left, rendered...)
}

// renderMessageBubble renders a single message bubble
func (m ChatModel) renderMessageBubble(msg storage.Message, isStreaming bool) string {
	content := msg.Content

	var bubble string
	if msg.Role == "user" {
		// User message: right-aligned, accent color
		userStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#04B575")).
			Foreground(lipgloss.Color("#04B575")).
			Padding(0, 1).
			MaxWidth(m.width - 20).
			Align(lipgloss.Right)

		bubble = userStyle.Render(content)

		// Right-align the whole bubble
		bubble = lipgloss.NewStyle().
			Width(m.width - 8).
			Align(lipgloss.Right).
			Render(bubble)

	} else {
		// Assistant message: left-aligned, markdown rendered
		// Render markdown
		rendered := m.renderMarkdown(content)

		assistantStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6c6c6c")).
			Foreground(lipgloss.Color("#e0e0e0")).
			Padding(0, 1).
			MaxWidth(m.width - 20).
			Align(lipgloss.Left)

		bubble = assistantStyle.Render(rendered)
	}

	// Add spacing between messages
	return bubble + "\n"
}

// renderMarkdown converts markdown to terminal-rendered output
func (m ChatModel) renderMarkdown(content string) string {
	// Calculate width for glamour (account for bubble border and padding)
	glamourWidth := m.width - 26

	if glamourWidth < 40 {
		glamourWidth = 40 // Minimum width
	}

	// Use DarkStyle instead of AutoStyle to avoid terminal color queries
	// AutoStyle queries the terminal which generates escape sequences that
	// can interfere with input
	r, err := glamour.NewTermRenderer(
		glamour.WithStylePath("dark"),
		glamour.WithWordWrap(glamourWidth),
	)
	if err != nil {
		return content // Fallback to plain text
	}

	rendered, err := r.Render(content)
	if err != nil {
		return content // Fallback to plain text
	}

	return strings.TrimSpace(rendered)
}

// retryLastMessage retries sending the last user message
func (m *ChatModel) retryLastMessage() tea.Cmd {
	// Find last user message
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == "user" {
			// Restart streaming
			m.isStreaming = true
			m.streamingContent = ""
			m.lastError = nil

			return m.llmClient.StreamResponse(context.Background(), m.messages, m.convContext(), m.topic.Mode)
		}
	}
	return nil
}

// convContext returns the conversation context for the system prompt
func (m ChatModel) convContext() llm.ConversationContext {
	return llm.ConversationContext{
		BookTitle:  m.book.Title,
		BookAuthor: m.book.Author,
		TopicName:  m.topic.Name,
	}
}

// calculateContextPercent calculates current context usage percentage
func (m ChatModel) calculateContextPercent() int {
	const maxTokens = 200000 // Claude 3.5/4 standard context window
	systemPrompt := llm.BuildSystemPrompt(m.convContext(), m.topic.Mode)
	currentTokens := storage.EstimateConversationTokens(systemPrompt, m.messages)
	percent := (currentTokens * 100) / maxTokens
	// Cap at 100 for display
	if percent > 100 {
		percent = 100
	}
	return percent
}

// checkAndSummarizeIfNeeded checks if context is over threshold and triggers summarization
func (m *ChatModel) checkAndSummarizeIfNeeded() tea.Cmd {
	const summarizeThreshold = 95
	const minMessagesForSummary = 10

	if m.contextPercent >= summarizeThreshold &&
		len(m.messages) >= minMessagesForSummary &&
		!m.isStreaming &&
		!m.hasSummary { // Only summarize once
		return func() tea.Msg { return summarizeStartMsg{} }
	}
	return nil
}

// summarizeOldMessagesCmd performs conversation summarization in background
func (m ChatModel) summarizeOldMessagesCmd() tea.Cmd {
	return func() tea.Msg {
		// Split: oldest 50% to archive, recent 50% to keep
		splitPoint := len(m.messages) / 2
		toSummarize := m.messages[:splitPoint]
		toKeep := m.messages[splitPoint:]

		// Archive original messages before summarization
		if err := storage.SaveChatArchive(m.topic.ChatFileUUID, toSummarize); err != nil {
			return summarizeErrorMsg{err: err}
		}

		// Build conversation text for summarization
		var conversationText strings.Builder
		for _, msg := range toSummarize {
			conversationText.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
		}

		summaryPrompt := fmt.Sprintf(`Summarize the following conversation between a user and assistant discussing "%s" by %s.

Focus on:
- Key ideas and arguments presented
- Intellectual conclusions or insights reached
- Important questions raised
- Any references to specific passages or concepts

Be concise but preserve the substance of the discussion. This summary will be used to maintain context in an ongoing conversation.

Conversation:
%s

Summary:`, m.book.Title, m.book.Author, conversationText.String())

		// Create temporary message list for summary generation
		summaryMessages := []storage.Message{
			{Role: "user", Content: summaryPrompt, Timestamp: time.Now().Unix()},
		}

		// Stream and accumulate summary (blocking call)
		ctx := context.Background()
		convCtx := m.convContext()

		summary := accumulateSummary(ctx, m.llmClient, summaryMessages, convCtx)

		// Build new message list: [summary message] + [recent messages]
		newMessages := []storage.Message{
			{
				Role:      "assistant",
				Content:   fmt.Sprintf("**[Summary of earlier conversation]**\n\n%s\n\n---\n", summary),
				Timestamp: time.Now().Unix(),
			},
		}
		newMessages = append(newMessages, toKeep...)

		return summarizeCompleteMsg{
			summary:  summary,
			messages: newMessages,
		}
	}
}

// accumulateSummary streams response from LLM and returns complete text
func accumulateSummary(ctx context.Context, client *llm.Client, messages []storage.Message, convCtx llm.ConversationContext) string {
	// Use a channel to accumulate streaming response
	type streamResult struct {
		text string
		err  error
	}

	resultCh := make(chan streamResult, 1)

	// Start streaming in goroutine
	go func() {
		var accumulated strings.Builder

		// Start stream
		stream := client.StreamResponse(ctx, messages, convCtx, "scholar")()

		// Process stream
		for {
			msg := stream.(llm.StreamStartMsg)
			for msg.Stream.Next() {
				event := msg.Stream.Current()
				if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
					accumulated.WriteString(event.Delta.Text)
				}
			}

			// Check for errors
			if err := msg.Stream.Err(); err != nil {
				resultCh <- streamResult{text: "", err: err}
				return
			}

			// Stream complete
			resultCh <- streamResult{text: accumulated.String(), err: nil}
			return
		}
	}()

	// Wait for result
	result := <-resultCh
	if result.err != nil {
		return fmt.Sprintf("Error generating summary: %v", result.err)
	}

	return result.text
}

// Commands

func (m ChatModel) loadChatCmd() tea.Cmd {
	return func() tea.Msg {
		messages, err := storage.LoadChat(m.topic.ChatFileUUID)
		if err != nil {
			// Return empty messages on error
			return chatLoadedMsg{messages: []storage.Message{}}
		}
		return chatLoadedMsg{messages: messages}
	}
}

func (m ChatModel) saveChatCmd() tea.Cmd {
	return func() tea.Msg {
		err := storage.SaveChat(m.topic.ChatFileUUID, m.messages)
		if err != nil {
			// In production, would return error message
			return nil
		}
		return chatSavedMsg{}
	}
}


func (m ChatModel) IsStreaming() bool {
	return m.isStreaming
}


func (m ChatModel) GetContextPercent() int {
	return m.contextPercent
}


func (m ChatModel) HasSummary() bool {
	return m.hasSummary
}


func (m ChatModel) GetMode() string {
	if m.topic.Mode == "" {
		return "scholar"
	}
	return m.topic.Mode
}

func (m ChatModel) updateTopicModeCmd(mode string) tea.Cmd {
	return func() tea.Msg {
		err := storage.UpdateTopicMode(m.db, m.topic.ID, mode)
		if err != nil {
			return nil
		}
		return updateTopicModeMsg{mode: mode}
	}
}

// Tool execution result message
type toolsExecutedMsg struct {
	toolCalls []llm.ToolCall
	results   []llm.ToolResultMsg
	convCtx   llm.ConversationContext
	mode      string
}

// executeToolsCmd executes tool calls and returns results
func (m ChatModel) executeToolsCmd(msg llm.StreamToolUseMsg) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var results []llm.ToolResultMsg

		convCtx := llm.ConversationContext{
			BookTitle:  m.book.Title,
			BookAuthor: m.book.Author,
			TopicName:  m.topic.Name,
		}

		// Check if research client is available
		researchClient, ok := msg.ResearchClient.(*research.SearchClient)
		if !ok || researchClient == nil {
			// Return error results for all tool calls
			for _, tc := range msg.ToolCalls {
				results = append(results, llm.ToolResultMsg{
					ToolID:  tc.ID,
					Content: "Research not available - Firecrawl API key not configured",
					IsError: true,
				})
			}
			return toolsExecutedMsg{
				toolCalls: msg.ToolCalls,
				results:   results,
				convCtx:   convCtx,
				mode:      msg.Mode,
			}
		}

		for _, tc := range msg.ToolCalls {
			result := llm.ExecuteToolCall(
				ctx,
				researchClient,
				m.book.Title,
				m.book.Author,
				tc.Name,
				tc.ID,
				tc.Input,
			)
			results = append(results, result)
		}

		return toolsExecutedMsg{
			toolCalls: msg.ToolCalls,
			results:   results,
			convCtx:   convCtx,
			mode:      msg.Mode,
		}
	}
}

// updateCommandHints filters slash commands based on current input
func (m *ChatModel) updateCommandHints() {
	content := m.input.Value()

	// Only show hints when input starts with /
	if !strings.HasPrefix(content, "/") {
		m.showCommandHints = false
		m.commandHints = nil
		m.selectedHint = 0
		return
	}

	// Filter commands that match the input prefix
	var hints []string
	for _, cmd := range slashCommands {
		if strings.HasPrefix(cmd, content) && cmd != content {
			hints = append(hints, cmd)
		}
	}

	m.commandHints = hints
	m.showCommandHints = len(hints) > 0

	// Reset selection if out of bounds
	if m.selectedHint >= len(hints) {
		m.selectedHint = 0
	}
}

// isExactCommand checks if input exactly matches a command
func isExactCommand(input string) bool {
	for _, cmd := range slashCommands {
		if input == cmd {
			return true
		}
	}
	return false
}


func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
