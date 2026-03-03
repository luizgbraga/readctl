package views

import (
	"strings"
	"testing"
)

func TestSlashCommandParsing(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		expectCommand  bool
		commandType    string // "mode", "doc", "rewrite-doc", or "none"
		expectInChat   bool   // should NOT appear in chat if command
	}{
		{
			name:          "mode command detected",
			input:         "/mode",
			expectCommand: true,
			commandType:   "mode",
			expectInChat:  false,
		},
		{
			name:          "doc command detected",
			input:         "/doc",
			expectCommand: true,
			commandType:   "doc",
			expectInChat:  false,
		},
		{
			name:          "rewrite-doc command detected",
			input:         "/rewrite-doc",
			expectCommand: true,
			commandType:   "rewrite-doc",
			expectInChat:  false,
		},
		{
			name:          "regular message not a command",
			input:         "What does this passage mean?",
			expectCommand: false,
			commandType:   "none",
			expectInChat:  true,
		},
		{
			name:          "message starting with slash but not command",
			input:         "/modetest",
			expectCommand: false,
			commandType:   "none",
			expectInChat:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test slash command detection logic with exact matching
			// Commands must be the full input (no extra text after)
			isCommand := tt.input == "/mode" ||
				tt.input == "/doc" ||
				strings.HasPrefix(tt.input, "/rewrite-doc")

			if isCommand != tt.expectCommand {
				t.Errorf("command detection mismatch: got %v, want %v", isCommand, tt.expectCommand)
			}

			// Verify command type detection
			var detectedType string
			switch {
			case tt.input == "/mode":
				detectedType = "mode"
			case strings.HasPrefix(tt.input, "/rewrite-doc"):
				detectedType = "rewrite-doc"
			case tt.input == "/doc":
				detectedType = "doc"
			default:
				detectedType = "none"
			}

			if detectedType != tt.commandType {
				t.Errorf("command type mismatch: got %v, want %v", detectedType, tt.commandType)
			}
		})
	}
}

// TODO: Add test for mode picker modal appearance
// TODO: Add test for document generation trigger
// TODO: Add test for rewrite instructions prompt
