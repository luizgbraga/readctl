package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/luizgbraga/readctl/internal/research"
	"github.com/luizgbraga/readctl/internal/storage"
)

// ToolUseMsg signals that Claude wants to use a tool
type ToolUseMsg struct {
	ToolID    string
	ToolName  string
	ToolInput json.RawMessage
}

// ToolResultMsg carries the result of a tool execution back to the LLM
type ToolResultMsg struct {
	ToolID  string
	Content string
	IsError bool
}

// ResearchConfig holds configuration for research capabilities
type ResearchConfig struct {
	FirecrawlAPIKey string
	Enabled         bool
}

func GetToolParams() []anthropic.ToolUnionParam {
	return []anthropic.ToolUnionParam{
		{OfTool: &anthropic.ToolParam{
			Name:        "search_web",
			Description: anthropic.String("Search the web for information about books, literary criticism, reviews, author biographies, and general context. Use this when you need current or specific information about a work, when you're uncertain about details, or when the user asks about something that requires external knowledge. Proactively search when discussing lesser-known works or when specificity would strengthen your response."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query. Be specific and include book title/author when relevant.",
					},
				},
				Required: []string{"query"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "search_academic",
			Description: anthropic.String("Search academic papers and scholarly articles for literary criticism, philosophical analysis, and peer-reviewed research. Use this to find scholarly interpretations, cite academic sources, and ground your analysis in published criticism. Essential for Scholar mode discussions."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The academic search query. Include author names, book titles, theoretical concepts, or critic names.",
					},
				},
				Required: []string{"query"},
			},
		}},
		{OfTool: &anthropic.ToolParam{
			Name:        "search_book_context",
			Description: anthropic.String("Search for specific information about the current book - excerpts, chapter analyses, themes, character studies, and critical interpretations. Use this to find relevant passages, verify quotes, or gather detailed analysis of specific aspects of the work being discussed."),
			InputSchema: anthropic.ToolInputSchemaParam{
				Properties: map[string]interface{}{
					"topic": map[string]interface{}{
						"type":        "string",
						"description": "The specific topic, theme, chapter, character, or aspect of the book to search for.",
					},
				},
				Required: []string{"topic"},
			},
		}},
	}
}

// ExecuteToolCall executes a tool call and returns the result
func ExecuteToolCall(ctx context.Context, researchClient *research.SearchClient, bookTitle, bookAuthor string, toolName string, toolID string, toolInput json.RawMessage) ToolResultMsg {
	var input map[string]string
	if err := json.Unmarshal(toolInput, &input); err != nil {
		return ToolResultMsg{
			ToolID:  toolID,
			Content: fmt.Sprintf("Error parsing tool input: %v", err),
			IsError: true,
		}
	}

	var results []research.SearchResult
	var err error

	switch toolName {
	case "search_web":
		query := input["query"]
		results, err = researchClient.SearchWeb(ctx, query, 5)

	case "search_academic":
		query := input["query"]
		results, err = researchClient.SearchAcademic(ctx, query, 5)

	case "search_book_context":
		topic := input["topic"]
		results, err = researchClient.SearchBookContext(ctx, bookTitle, bookAuthor, topic, 5)

	default:
		return ToolResultMsg{
			ToolID:  toolID,
			Content: fmt.Sprintf("Unknown tool: %s", toolName),
			IsError: true,
		}
	}

	if err != nil {
		return ToolResultMsg{
			ToolID:  toolID,
			Content: fmt.Sprintf("Search failed: %v", err),
			IsError: true,
		}
	}

	if len(results) == 0 {
		return ToolResultMsg{
			ToolID:  toolID,
			Content: "No results found for this query.",
		}
	}

	content := formatResultsForLLM(results)
	return ToolResultMsg{
		ToolID:  toolID,
		Content: content,
	}
}

func formatResultsForLLM(results []research.SearchResult) string {
	var output string
	for i, r := range results {
		output += fmt.Sprintf("**[%d] %s**\n", i+1, r.Title)
		if r.Authors != "" {
			output += fmt.Sprintf("Authors: %s\n", r.Authors)
		}
		if r.PublishedAt != "" {
			output += fmt.Sprintf("Published: %s\n", r.PublishedAt)
		}
		if r.Citations > 0 {
			output += fmt.Sprintf("Citations: %d\n", r.Citations)
		}
		output += fmt.Sprintf("Source: %s\n", r.URL)
		if r.Snippet != "" {
			output += fmt.Sprintf("\n%s\n", r.Snippet)
		}
		output += "\n---\n\n"
	}
	return output
}

func BuildMessagesWithToolResults(messages []storage.Message, toolCalls []ToolCall, toolResults []ToolResultMsg) []anthropic.MessageParam {
	params := messagesToParams(messages)

	if len(toolCalls) > 0 {
		// Add assistant message with tool_use blocks
		var toolUseBlocks []anthropic.ContentBlockParamUnion
		for _, tc := range toolCalls {
			// Parse the input JSON to pass as any
			var input any
			json.Unmarshal(tc.Input, &input)
			toolUseBlocks = append(toolUseBlocks, anthropic.NewToolUseBlock(tc.ID, input, tc.Name))
		}
		params = append(params, anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleAssistant,
			Content: toolUseBlocks,
		})
	}

	if len(toolResults) > 0 {
		// Add user message with tool_result blocks
		var toolResultBlocks []anthropic.ContentBlockParamUnion
		for _, tr := range toolResults {
			toolResultBlocks = append(toolResultBlocks, anthropic.NewToolResultBlock(tr.ToolID, tr.Content, tr.IsError))
		}
		params = append(params, anthropic.MessageParam{
			Role:    anthropic.MessageParamRoleUser,
			Content: toolResultBlocks,
		})
	}

	return params
}
