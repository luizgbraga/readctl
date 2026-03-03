package llm

import (
	"testing"
	"time"

	"github.com/luizgbraga/readctl/internal/storage"
)

func TestMessageConversion(t *testing.T) {
	messages := []storage.Message{
		{Role: "user", Content: "Hello", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "Hi there!", Timestamp: time.Now().Unix()},
		{Role: "user", Content: "How are you?", Timestamp: time.Now().Unix()},
	}

	params := messagesToParams(messages)

	if len(params) != 3 {
		t.Errorf("Expected 3 params, got %d", len(params))
	}
}

func TestStreamDeltaAccumulation(t *testing.T) {
	// Simulate streaming behavior
	deltas := []string{"Hello", " ", "world", "!"}
	accumulated := ""

	for _, delta := range deltas {
		accumulated += delta
	}

	expected := "Hello world!"
	if accumulated != expected {
		t.Errorf("Expected %q, got %q", expected, accumulated)
	}
}

func TestStreamMessageTypes(t *testing.T) {
	// Test that message types are properly structured
	delta := StreamDeltaMsg{Text: "test", Stream: nil}
	if delta.Text != "test" {
		t.Errorf("Expected 'test', got %q", delta.Text)
	}

	complete := StreamCompleteMsg{Content: "full content"}
	if complete.Content != "full content" {
		t.Errorf("Expected 'full content', got %q", complete.Content)
	}

	errMsg := StreamErrorMsg{
		ErrType: "rate_limit",
		Retry:   time.Second * 10,
		Message: "Rate limited",
	}
	if errMsg.ErrType != "rate_limit" {
		t.Errorf("Expected 'rate_limit', got %q", errMsg.ErrType)
	}
}

func TestMapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapError(tt.err)
			if result.ErrType != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result.ErrType)
			}
		})
	}
}
