package llm

import (
	"context"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	tea "charm.land/bubbletea/v2"
	"github.com/luizgbraga/readctl/internal/research"
	"github.com/luizgbraga/readctl/internal/storage"
)

type Client struct {
	client         anthropic.Client
	model          string
	researchClient *research.SearchClient
	researchEnabled bool
}

type ConversationContext struct {
	BookTitle  string
	BookAuthor string
	TopicName  string
}

func NewClient(apiKey, model, firecrawlAPIKey string) *Client {
	client := anthropic.NewClient(
		option.WithAPIKey(apiKey),
	)

	// Default to Claude Sonnet 4.5 if no model specified
	if model == "" {
		model = string(anthropic.ModelClaudeSonnet4_5_20250929)
	}

	var researchClient *research.SearchClient
	researchEnabled := firecrawlAPIKey != ""
	if researchEnabled {
		researchClient = research.NewSearchClient(firecrawlAPIKey)
	}

	return &Client{
		client:          client,
		model:           model,
		researchClient:  researchClient,
		researchEnabled: researchEnabled,
	}
}

func (c *Client) ResearchEnabled() bool {
	return c.researchEnabled
}

// Exported for use by chat view for context calculation
func BuildSystemPrompt(convCtx ConversationContext, mode string) string {
	basePrompt := fmt.Sprintf(`You are an intellectual companion for serious reading discussions. You are helping the user engage with "%s" by %s.

Current topic: %s

Your role:
- Engage thoughtfully with the text's ideas, themes, and arguments
- Draw on your knowledge of the work and its author
- Reference specific passages, chapters, or concepts when relevant
- Connect ideas to broader philosophical, literary, or historical context
- Challenge the user's interpretations constructively when appropriate
- Help clarify complex passages or concepts
- Track the thread of discussion within this topic

Source Attribution Guidelines:
- Distinguish between three types of knowledge:
  1. **Original text**: Direct content from "%s" — cite chapter/passage when you recall specifics
  2. **Literary criticism**: Scholarly interpretations or critical analyses — reference the critic or school of thought when relevant
  3. **Your interpretation**: Your own synthesis or analysis — present clearly as your reading
- Use natural language attribution (e.g., "In Chapter 3, X argues...", "Critics like Y have noted...", "My reading is...")
- Only attribute when ambiguous or when it adds intellectual value
- Avoid over-citing obvious points or universally known facts about the work

Style guidelines:
- Be substantive but concise — this is a terminal interface
- Use markdown formatting (bold, lists, code blocks for quotes)
- Avoid excessive hedging or disclaimers
- Engage as a knowledgeable peer, not a cautious assistant
- If you don't know something specific about the text, say so directly

Remember: The user chose this book deliberately and wants rigorous intellectual engagement, not summaries or surface-level discussion.`,
		convCtx.BookTitle,
		convCtx.BookAuthor,
		convCtx.TopicName,
		convCtx.BookTitle,
	)

	// Append mode-specific instructions
	modeInstructions := getModeInstructions(mode)
	return basePrompt + "\n\n" + modeInstructions
}

func getModeInstructions(mode string) string {
	switch mode {
	case "scholar":
		return `Mode: Scholar
Your emphasis is on academic depth. Draw on literary criticism, historical context, the author's biography, and intertextual references. Engage like a professor who brings deep scholarly knowledge to the conversation.`

	case "socratic":
		return `Mode: Socratic
Your emphasis is on questions with occasional anchors. Primarily ask clarifying questions and challenge assumptions to help the user think through their ideas. If the user is stuck, offer stepping stones to move forward, but prefer questions over answers.`

	case "dialectical":
		return `Mode: Dialectical
Your emphasis is on steelman opposition. Construct the strongest possible counterargument to the user's position, engaging their ideas rigorously without necessarily naming a specific philosophical school.`

	case "provocateur":
		return `Mode: Provocateur
Your emphasis is sharp but fair. Points out weaknesses directly and challenges assumptions, but remains constructive. You are not playing devil's advocate or being intentionally antagonistic — you are helping the user strengthen their thinking through direct challenge.`

	default:
		// Default to scholar mode
		return getModeInstructions("scholar")
	}
}

func (c *Client) StreamResponse(ctx context.Context, messages []storage.Message, convCtx ConversationContext, mode string) tea.Cmd {
	return c.StreamResponseWithToolResults(ctx, messages, convCtx, mode, nil, nil)
}

func (c *Client) StreamResponseWithToolResults(ctx context.Context, messages []storage.Message, convCtx ConversationContext, mode string, toolCalls []ToolCall, toolResults []ToolResultMsg) tea.Cmd {
	return func() tea.Msg {
		// Convert storage messages to API params
		var params []anthropic.MessageParam
		if len(toolResults) > 0 {
			params = BuildMessagesWithToolResults(messages, toolCalls, toolResults)
		} else {
			params = messagesToParams(messages)
		}

		// Build system prompt with context and mode
		systemPrompt := BuildSystemPrompt(convCtx, mode)

		// Build request params
		reqParams := anthropic.MessageNewParams{
			Model:     anthropic.Model(c.model),
			MaxTokens: 4096,
			System:    []anthropic.TextBlockParam{{Text: systemPrompt}},
			Messages:  params,
		}

		// Add tools if research is enabled
		if c.researchEnabled {
			reqParams.Tools = GetToolParams()
		}

		// Create streaming request
		stream := c.client.Messages.NewStreaming(ctx, reqParams)

		// Return a batch command that will process stream events
		return StreamStartMsg{
			Stream:          stream,
			ConvCtx:         convCtx,
			Mode:            mode,
			ResearchClient:  c.researchClient,
			ResearchEnabled: c.researchEnabled,
		}
	}
}

// StreamProcessor handles the complex logic of processing stream events including tool use
type StreamProcessor struct {
	Stream          *Stream
	ConvCtx         ConversationContext
	Mode            string
	ResearchClient  *research.SearchClient
	ResearchEnabled bool
	ToolCalls       []ToolCall
	CurrentToolID   string
	CurrentToolName string
	CurrentToolInput []byte
}

func NewStreamProcessor(msg StreamStartMsg) *StreamProcessor {
	var rc *research.SearchClient
	if msg.ResearchClient != nil {
		rc = msg.ResearchClient.(*research.SearchClient)
	}
	return &StreamProcessor{
		Stream:          msg.Stream,
		ConvCtx:         msg.ConvCtx,
		Mode:            msg.Mode,
		ResearchClient:  rc,
		ResearchEnabled: msg.ResearchEnabled,
	}
}

func (p *StreamProcessor) ProcessNext() tea.Cmd {
	return func() tea.Msg {
		if p.Stream.Next() {
			event := p.Stream.Current()

			switch event.Type {
			case "content_block_start":
				// Check if starting a tool use block
				if event.ContentBlock.Type == "tool_use" {
					p.CurrentToolID = event.ContentBlock.ID
					p.CurrentToolName = event.ContentBlock.Name
					p.CurrentToolInput = nil
				}

			case "content_block_delta":
				if event.Delta.Type == "text_delta" {
					return StreamDeltaMsg{
						Text:   event.Delta.Text,
						Stream: p.Stream,
					}
				} else if event.Delta.Type == "input_json_delta" {
					// Accumulate tool input JSON
					p.CurrentToolInput = append(p.CurrentToolInput, []byte(event.Delta.PartialJSON)...)
				}

			case "content_block_stop":
				// If we were building a tool call, save it
				if p.CurrentToolID != "" {
					p.ToolCalls = append(p.ToolCalls, ToolCall{
						ID:    p.CurrentToolID,
						Name:  p.CurrentToolName,
						Input: p.CurrentToolInput,
					})
					p.CurrentToolID = ""
					p.CurrentToolName = ""
					p.CurrentToolInput = nil
				}

			case "message_stop":
				// Check if we have tool calls to execute
				if len(p.ToolCalls) > 0 {
					return StreamToolUseMsg{
						ToolCalls:       p.ToolCalls,
						ConvCtx:         p.ConvCtx,
						Mode:            p.Mode,
						ResearchClient:  p.ResearchClient,
						ResearchEnabled: p.ResearchEnabled,
					}
				}
			}

			// Continue processing
			return StreamContinueMsg{
				Stream:          p.Stream,
				ConvCtx:         p.ConvCtx,
				Mode:            p.Mode,
				ResearchClient:  p.ResearchClient,
				ResearchEnabled: p.ResearchEnabled,
			}
		}

		// Stream has ended, check for errors
		if err := p.Stream.Err(); err != nil {
			return mapError(err)
		}

		// Check if we accumulated tool calls
		if len(p.ToolCalls) > 0 {
			return StreamToolUseMsg{
				ToolCalls:       p.ToolCalls,
				ConvCtx:         p.ConvCtx,
				Mode:            p.Mode,
				ResearchClient:  p.ResearchClient,
				ResearchEnabled: p.ResearchEnabled,
			}
		}

		// Stream complete - content already accumulated via StreamDeltaMsg
		return StreamCompleteMsg{Content: ""}
	}
}

// Legacy function for compatibility
func ProcessStreamEvent(stream *Stream) tea.Cmd {
	return func() tea.Msg {
		if stream.Next() {
			event := stream.Current()

			// Check if this is a content block delta with text
			if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
				return StreamDeltaMsg{
					Text:   event.Delta.Text,
					Stream: stream,
				}
			}

			// Continue processing other events
			return StreamContinueMsg{Stream: stream}
		}

		// Stream has ended, check for errors
		if err := stream.Err(); err != nil {
			return mapError(err)
		}

		// Stream complete - content already accumulated via StreamDeltaMsg
		return StreamCompleteMsg{Content: ""}
	}
}
