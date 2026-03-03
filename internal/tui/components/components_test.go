package components

import (
	"strings"
	"testing"
)

func TestRenderHeader(t *testing.T) {
	header := RenderHeader("Books", 80)
	if header == "" {
		t.Error("RenderHeader returned empty string")
	}
}

func TestRenderFooter(t *testing.T) {
	footer := RenderFooter("q: quit", 80)
	if footer == "" {
		t.Error("RenderFooter returned empty string")
	}
}

func TestHelpModel(t *testing.T) {
	help := NewHelpModel()
	help.SetKeys(BooksKeys())
	view := help.View(80, 24)
	if view == "" {
		t.Error("Help view returned empty string")
	}
}

func TestModalModel(t *testing.T) {
	modal := NewModalModel()

	// Initially not visible
	if modal.IsVisible() {
		t.Error("Modal should not be visible initially")
	}

	// Show modal
	modal.Show("Test", "Test message")
	if !modal.IsVisible() {
		t.Error("Modal should be visible after Show()")
	}

	view := modal.View(80, 24)
	if view == "" {
		t.Error("Modal view returned empty string")
	}

	// Hide modal
	modal.Hide()
	if modal.IsVisible() {
		t.Error("Modal should not be visible after Hide()")
	}
}

func TestBooksKeys(t *testing.T) {
	keys := BooksKeys()
	if len(keys) == 0 {
		t.Error("BooksKeys returned empty slice")
	}
}

func TestTopicsKeys(t *testing.T) {
	keys := TopicsKeys()
	if len(keys) == 0 {
		t.Error("TopicsKeys returned empty slice")
	}
}

func TestRenderFooterWithContext(t *testing.T) {
	tests := []struct {
		name           string
		hints          string
		contextPercent int
		width          int
		expectSep      bool   // Should contain " | "
		expectColor    string // Expected color range indicator
	}{
		{
			name:           "50% context shows gray",
			hints:          "q: quit",
			contextPercent: 50,
			width:          80,
			expectSep:      true,
			expectColor:    "gray", // 0-79% range
		},
		{
			name:           "80% context shows yellow",
			hints:          "q: quit",
			contextPercent: 80,
			width:          80,
			expectSep:      true,
			expectColor:    "yellow", // 80-94% range
		},
		{
			name:           "95% context shows red",
			hints:          "q: quit",
			contextPercent: 95,
			width:          80,
			expectSep:      true,
			expectColor:    "red", // 95-100% range
		},
		{
			name:           "120% context caps at 100",
			hints:          "q: quit",
			contextPercent: 120,
			width:          80,
			expectSep:      true,
			expectColor:    "red", // Caps at 100%
		},
		{
			name:           "0% context shows gray",
			hints:          "q: quit",
			contextPercent: 0,
			width:          80,
			expectSep:      true,
			expectColor:    "gray",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RenderFooterWithContext(tt.hints, tt.contextPercent, tt.width)

			if result == "" {
				t.Error("RenderFooterWithContext returned empty string")
			}

			// Check for separator
			if tt.expectSep && !strings.Contains(result, "|") {
				t.Errorf("Expected separator '|' in result")
			}

			// Check that percentage is displayed (even if capped)
			if !strings.Contains(result, "Context:") {
				t.Error("Expected 'Context:' in result")
			}

			// For >100%, verify it's capped
			if tt.contextPercent > 100 && strings.Contains(result, "120") {
				t.Error("Expected percentage to be capped at 100, but found 120")
			}
		})
	}
}
