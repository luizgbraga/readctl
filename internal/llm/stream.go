package llm

import (
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/packages/ssestream"
	"github.com/luizgbraga/readctl/internal/storage"
)

type Stream = ssestream.Stream[anthropic.MessageStreamEventUnion]

type StreamStartMsg struct {
	Stream          *Stream
	ConvCtx         ConversationContext
	Mode            string
	ResearchClient  interface{} // *research.SearchClient, using interface to avoid import cycle
	ResearchEnabled bool
}

type StreamDeltaMsg struct {
	Text   string
	Stream *Stream
}

type StreamContinueMsg struct {
	Stream          *Stream
	ConvCtx         ConversationContext
	Mode            string
	ResearchClient  interface{}
	ResearchEnabled bool
}

type StreamCompleteMsg struct {
	Content string
}

// StreamToolUseMsg signals that Claude wants to use tools before responding
type StreamToolUseMsg struct {
	ToolCalls       []ToolCall
	ConvCtx         ConversationContext
	Mode            string
	ResearchClient  interface{}
	ResearchEnabled bool
}

type ToolCall struct {
	ID    string
	Name  string
	Input []byte
}

type StreamErrorMsg struct {
	ErrType string
	Retry   time.Duration
	Message string
}

func messagesToParams(messages []storage.Message) []anthropic.MessageParam {
	params := make([]anthropic.MessageParam, 0, len(messages))

	for _, msg := range messages {
		var param anthropic.MessageParam

		if msg.Role == "user" {
			param = anthropic.NewUserMessage(
				anthropic.NewTextBlock(msg.Content),
			)
		} else if msg.Role == "assistant" {
			param = anthropic.NewAssistantMessage(
				anthropic.NewTextBlock(msg.Content),
			)
		}

		params = append(params, param)
	}

	return params
}

func mapError(err error) StreamErrorMsg {
	if err == nil {
		return StreamErrorMsg{}
	}

	// Check error message for rate limiting
	errMsg := strings.ToLower(err.Error())
	if strings.Contains(errMsg, "rate limit") || strings.Contains(errMsg, "429") {
		return StreamErrorMsg{
			ErrType: "rate_limit",
			Retry:   time.Second * 10,
			Message: "API rate limited. Retrying...",
		}
	}

	// Check for authentication errors
	if strings.Contains(errMsg, "authentication") || strings.Contains(errMsg, "401") {
		return StreamErrorMsg{
			ErrType: "auth",
			Message: "Authentication failed. Check API key.",
		}
	}

	// Default to showing the actual error
	return StreamErrorMsg{
		ErrType: "error",
		Message: err.Error(),
	}
}
