package research

import (
	"context"
	"encoding/json"
	"fmt"
)

// ToolDefinition represents a tool that Claude can use
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// GetTools returns the tool definitions for Claude
func GetTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Name:        "search_web",
			Description: "Search the web for information about books, literary criticism, reviews, author biographies, and general context. Use this when the user asks about something you're not certain about, or when you need current/specific information about a work.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The search query. Be specific and include book title/author when relevant.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "search_academic",
			Description: "Search academic papers and scholarly articles for literary criticism, philosophical analysis, and peer-reviewed research about books and authors. Use this for finding scholarly interpretations and academic perspectives.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "The academic search query. Include author names, book titles, or theoretical concepts.",
					},
				},
				"required": []string{"query"},
			},
		},
		{
			Name:        "search_book_context",
			Description: "Search for specific information about the current book being discussed - excerpts, chapter analyses, themes, character studies, and critical interpretations. This is optimized for finding literary analysis of the specific work.",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"topic": map[string]interface{}{
						"type":        "string",
						"description": "The specific topic, theme, chapter, or aspect of the book to search for.",
					},
				},
				"required": []string{"topic"},
			},
		},
	}
}

// ToolCall represents a tool invocation from Claude
type ToolCall struct {
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// ExecuteTool runs a tool and returns the result
func ExecuteTool(ctx context.Context, client *SearchClient, bookTitle, bookAuthor string, call ToolCall) ToolResult {
	var input map[string]string
	if err := json.Unmarshal(call.Input, &input); err != nil {
		return ToolResult{
			ToolUseID: call.ID,
			Content:   fmt.Sprintf("Error parsing tool input: %v", err),
			IsError:   true,
		}
	}

	var results []SearchResult
	var err error

	switch call.Name {
	case "search_web":
		query := input["query"]
		results, err = client.SearchWeb(ctx, query, 5)

	case "search_academic":
		query := input["query"]
		results, err = client.SearchAcademic(ctx, query, 5)

	case "search_book_context":
		topic := input["topic"]
		results, err = client.SearchBookContext(ctx, bookTitle, bookAuthor, topic, 5)

	default:
		return ToolResult{
			ToolUseID: call.ID,
			Content:   fmt.Sprintf("Unknown tool: %s", call.Name),
			IsError:   true,
		}
	}

	if err != nil {
		return ToolResult{
			ToolUseID: call.ID,
			Content:   fmt.Sprintf("Search failed: %v", err),
			IsError:   true,
		}
	}

	if len(results) == 0 {
		return ToolResult{
			ToolUseID: call.ID,
			Content:   "No results found.",
		}
	}

	// Format results for Claude
	content := formatResultsForLLM(results)
	return ToolResult{
		ToolUseID: call.ID,
		Content:   content,
	}
}

func formatResultsForLLM(results []SearchResult) string {
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
