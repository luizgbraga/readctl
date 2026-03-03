package storage

import (
	"unicode/utf8"
)

// EstimateTokens estimates the number of tokens in a text string using
// the heuristic of 1 token ≈ 4 characters (Anthropic guidance).
// Uses rune count (not byte count) to handle UTF-8 correctly.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	runeCount := utf8.RuneCountInString(text)
	// Conservative rounding: (count + 3) / 4 rounds up
	return (runeCount + 3) / 4
}

// EstimateConversationTokens estimates total tokens for a conversation,
// including system prompt and all messages.
func EstimateConversationTokens(systemPrompt string, messages []Message) int {
	total := EstimateTokens(systemPrompt)

	for _, msg := range messages {
		total += EstimateTokens(msg.Content)
	}

	return total
}
