package storage

import (
	"testing"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "empty string",
			input:    "",
			expected: 0,
		},
		{
			name:     "simple ASCII text",
			input:    "hello",
			expected: 2, // 5 chars / 4 = 1.25, rounds up to 2
		},
		{
			name:     "emoji single character",
			input:    "🚀",
			expected: 1, // 1 rune / 4 = 0.25, rounds up to 1
		},
		{
			name:     "multi-byte UTF-8 characters",
			input:    "こんにちは", // 5 Japanese characters
			expected: 2,           // 5 runes / 4 = 1.25, rounds up to 2
		},
		{
			name:     "longer text",
			input:    "The quick brown fox jumps over the lazy dog",
			expected: 11, // 44 chars / 4 = 11
		},
		{
			name:     "mixed ASCII and emoji",
			input:    "Hello 👋 World 🌍",
			expected: 4, // 15 runes / 4 = 3.75, rounds up to 4
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateTokens(tt.input)
			if result != tt.expected {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestEstimateConversationTokens(t *testing.T) {
	tests := []struct {
		name         string
		systemPrompt string
		messages     []Message
		expected     int
	}{
		{
			name:         "empty conversation",
			systemPrompt: "",
			messages:     []Message{},
			expected:     0,
		},
		{
			name:         "system prompt only",
			systemPrompt: "You are a helpful assistant",
			messages:     []Message{},
			expected:     7, // 28 chars / 4 = 7
		},
		{
			name:         "system prompt with one message",
			systemPrompt: "Assistant",
			messages: []Message{
				{Role: "user", Content: "Hello"},
			},
			expected: 5, // "Assistant" (9 chars / 4 = 2.25 -> 3) + "Hello" (5 chars / 4 = 1.25 -> 2) = 5
		},
		{
			name:         "full conversation",
			systemPrompt: "You are a book discussion assistant",
			messages: []Message{
				{Role: "user", Content: "What is the theme?"},
				{Role: "assistant", Content: "The theme explores identity and belonging."},
			},
			expected: 25, // system (36/4=9) + user (18/4=5) + assistant (42/4=11) = 25
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateConversationTokens(tt.systemPrompt, tt.messages)
			if result != tt.expected {
				t.Errorf("EstimateConversationTokens() = %d, want %d", result, tt.expected)
			}
		})
	}
}
