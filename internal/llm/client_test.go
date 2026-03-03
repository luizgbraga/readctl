package llm

import (
	"strings"
	"testing"
)

func TestModePrompts_Scholar(t *testing.T) {
	ctx := ConversationContext{
		BookTitle:  "Test Book",
		BookAuthor: "Test Author",
		TopicName:  "Test Topic",
	}

	prompt := BuildSystemPrompt(ctx, "scholar")

	// Scholar mode should include academic depth and literary criticism references
	expectedPhrases := []string{
		"academic depth",
		"literary criticism",
		"professor",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(phrase)) {
			t.Errorf("Scholar mode prompt should contain '%s', but doesn't", phrase)
		}
	}
}

func TestModePrompts_Socratic(t *testing.T) {
	ctx := ConversationContext{
		BookTitle:  "Test Book",
		BookAuthor: "Test Author",
		TopicName:  "Test Topic",
	}

	prompt := BuildSystemPrompt(ctx, "socratic")

	// Socratic mode should emphasize questions and stepping stones
	expectedPhrases := []string{
		"clarifying questions",
		"challenge assumptions",
		"stepping stones",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(phrase)) {
			t.Errorf("Socratic mode prompt should contain '%s', but doesn't", phrase)
		}
	}
}

func TestModePrompts_Dialectical(t *testing.T) {
	ctx := ConversationContext{
		BookTitle:  "Test Book",
		BookAuthor: "Test Author",
		TopicName:  "Test Topic",
	}

	prompt := BuildSystemPrompt(ctx, "dialectical")

	// Dialectical mode should emphasize counterarguments and steelmanning
	expectedPhrases := []string{
		"steelman",
		"counterargument",
		"strongest possible",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(phrase)) {
			t.Errorf("Dialectical mode prompt should contain '%s', but doesn't", phrase)
		}
	}
}

func TestModePrompts_Provocateur(t *testing.T) {
	ctx := ConversationContext{
		BookTitle:  "Test Book",
		BookAuthor: "Test Author",
		TopicName:  "Test Topic",
	}

	prompt := BuildSystemPrompt(ctx, "provocateur")

	// Provocateur mode should emphasize directness and constructive challenge
	expectedPhrases := []string{
		"sharp but fair",
		"points out weaknesses",
		"constructive",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(strings.ToLower(prompt), strings.ToLower(phrase)) {
			t.Errorf("Provocateur mode prompt should contain '%s', but doesn't", phrase)
		}
	}

	// Should clarify it's NOT antagonistic (mentions what it's NOT)
	// We expect the prompt to clarify the boundaries
	if !strings.Contains(strings.ToLower(prompt), "not") {
		t.Error("Provocateur mode should clarify what it is NOT (e.g., 'not devil's advocate')")
	}
}

func TestModePrompts_DefaultsToScholar(t *testing.T) {
	ctx := ConversationContext{
		BookTitle:  "Test Book",
		BookAuthor: "Test Author",
		TopicName:  "Test Topic",
	}

	tests := []struct {
		name string
		mode string
	}{
		{"empty mode", ""},
		{"unknown mode", "unknown"},
		{"invalid mode", "invalid_mode"},
	}

	scholarPrompt := BuildSystemPrompt(ctx, "scholar")

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := BuildSystemPrompt(ctx, tt.mode)

			// Should produce the same prompt as scholar mode
			if prompt != scholarPrompt {
				t.Errorf("Mode '%s' should default to scholar, but produced different prompt", tt.mode)
			}
		})
	}
}

func TestModePrompts_AllModesDistinct(t *testing.T) {
	ctx := ConversationContext{
		BookTitle:  "Test Book",
		BookAuthor: "Test Author",
		TopicName:  "Test Topic",
	}

	modes := []string{"scholar", "socratic", "dialectical", "provocateur"}
	prompts := make(map[string]string)

	// Generate all prompts
	for _, mode := range modes {
		prompts[mode] = BuildSystemPrompt(ctx, mode)
	}

	// Verify each prompt is distinct from the others
	for i, mode1 := range modes {
		for j, mode2 := range modes {
			if i != j && prompts[mode1] == prompts[mode2] {
				t.Errorf("Modes '%s' and '%s' produce identical prompts — they should be distinct", mode1, mode2)
			}
		}
	}
}

func TestModePrompts_BasePromptPreserved(t *testing.T) {
	ctx := ConversationContext{
		BookTitle:  "The Great Gatsby",
		BookAuthor: "F. Scott Fitzgerald",
		TopicName:  "American Dream",
	}

	modes := []string{"scholar", "socratic", "dialectical", "provocateur"}

	// All modes should still include base context elements
	baseElements := []string{
		"The Great Gatsby",
		"F. Scott Fitzgerald",
		"American Dream",
		"intellectual companion",
	}

	for _, mode := range modes {
		prompt := BuildSystemPrompt(ctx, mode)

		for _, element := range baseElements {
			if !strings.Contains(prompt, element) {
				t.Errorf("Mode '%s' prompt is missing base element '%s'", mode, element)
			}
		}
	}
}
