package views

import (
	"strings"
	"testing"

	"github.com/luizgbraga/readctl/internal/storage"
)

func TestDocumentEmptyState(t *testing.T) {
	topic := storage.Topic{
		ID:           1,
		BookID:       1,
		Name:         "Test Topic",
		ChatFileUUID: "nonexistent-uuid",
	}

	// Create document model
	model := NewDocumentModel(topic)
	model.width = 80
	model.height = 24

	// Render empty document
	rendered := model.renderDocument()

	// Verify empty state message
	expectedPhrases := []string{
		"No document yet",
		"/doc",
		"generate",
	}

	for _, phrase := range expectedPhrases {
		if !strings.Contains(rendered, phrase) {
			t.Errorf("empty state message missing phrase: %s", phrase)
		}
	}
}

func TestDocumentRendering(t *testing.T) {
	tests := []struct {
		name            string
		documentContent string
		expectEmpty     bool
	}{
		{
			name:            "renders markdown headers",
			documentContent: "# Title\n## Subtitle\nContent here",
			expectEmpty:     false,
		},
		{
			name:            "renders lists",
			documentContent: "- Point 1\n- Point 2\n- Point 3",
			expectEmpty:     false,
		},
		{
			name:            "empty content shows empty state",
			documentContent: "",
			expectEmpty:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			topic := storage.Topic{
				ID:           1,
				BookID:       1,
				Name:         "Test Topic",
				ChatFileUUID: "test-uuid",
			}

			model := NewDocumentModel(topic)
			model.width = 80
			model.height = 24
			model.content = tt.documentContent

			rendered := model.renderDocument()

			if tt.expectEmpty {
				if !strings.Contains(rendered, "No document yet") {
					t.Errorf("Expected empty state message, got: %s", rendered)
				}
			} else {
				if strings.Contains(rendered, "No document yet") {
					t.Errorf("Expected rendered content, got empty state")
				}
				if rendered == "" {
					t.Errorf("Expected non-empty rendered output")
				}
			}
		})
	}
}

// TODO: Add test for document loading from storage
// TODO: Add test for viewport scrolling behavior
// TODO: Add test for esc key returning to topics
