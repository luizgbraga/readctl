package views

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/luizgbraga/readctl/internal/storage"
)

// generateDocumentCmd generates a conclusion document from the conversation
func (m ChatModel) generateDocumentCmd() tea.Cmd {
	return func() tea.Msg {
		// Build conversation text from messages
		var conversationText strings.Builder
		for _, msg := range m.messages {
			conversationText.WriteString(fmt.Sprintf("%s: %s\n\n", msg.Role, msg.Content))
		}

		// Create document synthesis prompt
		prompt := fmt.Sprintf(`Synthesize the following conversation about "%s" by %s into a conclusion document.

Your task: Analyze the conversation and create a document with whatever structure best captures the discussion. This might be:
- A thematic essay if the conversation explored ideas systematically
- A Q&A format if it was exploratory
- A reference guide if specific concepts were discussed
- Annotated quotes if textual analysis was central
- Any other structure that serves the content

Guidelines:
- Choose the structure that best serves the intellectual content
- Preserve the substance and depth of the discussion
- Use markdown formatting (headers, lists, bold, quotes)
- Be substantive but focused - this is a reference document
- Aim for ~500-1000 words unless the conversation warrants more

Conversation:
%s

Generate the conclusion document now:`, m.book.Title, m.book.Author, conversationText.String())

		// Create temporary message list for document generation
		docMessages := []storage.Message{
			{Role: "user", Content: prompt, Timestamp: time.Now().Unix()},
		}

		// Stream and accumulate document (blocking call)
		ctx := context.Background()
		convCtx := m.convContext()

		content := accumulateSummary(ctx, m.llmClient, docMessages, convCtx)

		return documentGeneratedMsg{
			content: content,
			uuid:    m.topic.ChatFileUUID,
		}
	}
}

// rewriteDocumentCmd regenerates document with user instructions
func (m ChatModel) rewriteDocumentCmd(instructions string) tea.Cmd {
	return func() tea.Msg {
		// Load existing document
		existingDoc, err := storage.LoadDocument(m.topic.ChatFileUUID)
		if err != nil || existingDoc == "" {
			return documentGeneratedMsg{
				content: "Error: No existing document to rewrite",
				uuid:    m.topic.ChatFileUUID,
			}
		}

		// Create rewrite prompt
		prompt := fmt.Sprintf(`Here is an existing conclusion document:

%s

User requests the following changes:
%s

Rewrite the document incorporating these changes while maintaining its overall quality and structure. Use markdown formatting.`, existingDoc, instructions)

		// Create temporary message list for rewrite generation
		docMessages := []storage.Message{
			{Role: "user", Content: prompt, Timestamp: time.Now().Unix()},
		}

		// Stream and accumulate document (blocking call)
		ctx := context.Background()
		convCtx := m.convContext()

		content := accumulateSummary(ctx, m.llmClient, docMessages, convCtx)

		return documentGeneratedMsg{
			content: content,
			uuid:    m.topic.ChatFileUUID,
		}
	}
}

// handleDocumentCommand processes /doc and /rewrite-doc commands
// Returns true if command was handled, along with any tea.Cmd to execute
func (m *ChatModel) handleDocumentCommand(content string) (bool, tea.Cmd) {
	// Handle confirmation for document overwrite
	if m.confirmingDocOverwrite {
		if content == "y" || content == "Y" {
			m.confirmingDocOverwrite = false
			m.generatingDocument = true
			m.input.Reset()
			m.input.Placeholder = "Type a message..."
			if m.ready {
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
			}
			return true, m.generateDocumentCmd()
		} else if content == "n" || content == "N" {
			m.confirmingDocOverwrite = false
			m.input.Reset()
			m.input.Placeholder = "Type a message..."
			return true, nil
		}
		// Invalid input - keep waiting
		return true, nil
	}

	// Handle /doc command (exact match)
	if content == "/doc" {
		if m.documentExists {
			// Show confirmation
			m.confirmingDocOverwrite = true
			m.input.Reset()
			m.input.Placeholder = "Document exists. Overwrite? (y/n)"
			if m.ready {
				m.viewport.SetContent(m.renderMessages())
				m.viewport.GotoBottom()
			}
			return true, nil
		}
		// No existing document, generate immediately
		m.generatingDocument = true
		m.input.Reset()
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return true, m.generateDocumentCmd()
	}

	// Handle /rewrite-doc command
	if strings.HasPrefix(content, "/rewrite-doc") {
		m.awaitingRewriteInstructions = true
		m.input.Reset()
		m.input.Placeholder = "What changes do you want?"
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return true, nil
	}

	// Handle rewrite instructions
	if m.awaitingRewriteInstructions {
		instructions := content
		m.awaitingRewriteInstructions = false
		m.generatingDocument = true
		m.input.Reset()
		m.input.Placeholder = "Type a message..."
		if m.ready {
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return true, m.rewriteDocumentCmd(instructions)
	}

	return false, nil
}
