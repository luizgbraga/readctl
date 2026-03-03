package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// setupTestDB creates a temporary test database
func setupTestDB(t *testing.T) *sql.DB {
	// Create temp directory for test database
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("Failed to open test database: %v", err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		t.Fatalf("Failed to initialize schema: %v", err)
	}

	// Cleanup
	t.Cleanup(func() {
		db.Close()
		os.RemoveAll(tmpDir)
	})

	return db
}

// createTestBook creates a test book and returns its ID
func createTestBook(t *testing.T, db *sql.DB) int {
	result, err := db.Exec(
		"INSERT INTO books (title, author, created_at) VALUES (?, ?, ?)",
		"Test Book", "Test Author", 1000,
	)
	if err != nil {
		t.Fatalf("Failed to create test book: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get book ID: %v", err)
	}

	return int(id)
}

func TestCreateTopicWithMode(t *testing.T) {
	db := setupTestDB(t)
	bookID := createTestBook(t, db)

	tests := []struct {
		name         string
		mode         string
		expectedMode string
	}{
		{
			name:         "Scholar mode",
			mode:         "scholar",
			expectedMode: "scholar",
		},
		{
			name:         "Socratic mode",
			mode:         "socratic",
			expectedMode: "socratic",
		},
		{
			name:         "Dialectical mode",
			mode:         "dialectical",
			expectedMode: "dialectical",
		},
		{
			name:         "Provocateur mode",
			mode:         "provocateur",
			expectedMode: "provocateur",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create topic with mode
			topicID, chatUUID, err := CreateTopic(db, bookID, "Test Topic", tt.mode)
			if err != nil {
				t.Fatalf("CreateTopic failed: %v", err)
			}

			if topicID == 0 {
				t.Error("Expected non-zero topic ID")
			}

			if chatUUID == "" {
				t.Error("Expected non-empty chat UUID")
			}

			// Verify mode was saved
			var savedMode string
			err = db.QueryRow("SELECT mode FROM topics WHERE id = ?", topicID).Scan(&savedMode)
			if err != nil {
				t.Fatalf("Failed to query mode: %v", err)
			}

			if savedMode != tt.expectedMode {
				t.Errorf("Expected mode %s, got %s", tt.expectedMode, savedMode)
			}
		})
	}
}

func TestCreateTopicDefaultMode(t *testing.T) {
	db := setupTestDB(t)
	bookID := createTestBook(t, db)

	tests := []struct {
		name         string
		mode         string
		expectedMode string
	}{
		{
			name:         "Empty mode defaults to scholar",
			mode:         "",
			expectedMode: "scholar",
		},
		{
			name:         "Whitespace mode defaults to scholar",
			mode:         "   ",
			expectedMode: "scholar",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create topic with empty/whitespace mode
			topicID, _, err := CreateTopic(db, bookID, "Test Topic", tt.mode)
			if err != nil {
				t.Fatalf("CreateTopic failed: %v", err)
			}

			// Verify mode defaulted to scholar
			var savedMode string
			err = db.QueryRow("SELECT mode FROM topics WHERE id = ?", topicID).Scan(&savedMode)
			if err != nil {
				t.Fatalf("Failed to query mode: %v", err)
			}

			if savedMode != tt.expectedMode {
				t.Errorf("Expected default mode %s, got %s", tt.expectedMode, savedMode)
			}
		})
	}
}

func TestGetTopicsIncludesMode(t *testing.T) {
	db := setupTestDB(t)
	bookID := createTestBook(t, db)

	// Create topics with different modes
	modes := []string{"scholar", "socratic", "dialectical", "provocateur"}
	for i, mode := range modes {
		_, _, err := CreateTopic(db, bookID, "Topic "+string(rune('A'+i)), mode)
		if err != nil {
			t.Fatalf("Failed to create topic with mode %s: %v", mode, err)
		}
	}

	// Get all topics
	topics, err := GetTopics(db, bookID)
	if err != nil {
		t.Fatalf("GetTopics failed: %v", err)
	}

	if len(topics) != len(modes) {
		t.Errorf("Expected %d topics, got %d", len(modes), len(topics))
	}

	// Verify all topics have mode field populated
	for _, topic := range topics {
		if topic.Mode == "" {
			t.Errorf("Topic %d has empty mode field", topic.ID)
		}

		// Verify mode is one of the expected values
		validMode := false
		for _, mode := range modes {
			if topic.Mode == mode {
				validMode = true
				break
			}
		}
		if !validMode {
			t.Errorf("Topic %d has unexpected mode: %s", topic.ID, topic.Mode)
		}
	}
}

func TestExistingTopicsGetDefaultMode(t *testing.T) {
	db := setupTestDB(t)
	bookID := createTestBook(t, db)

	// Insert topic WITHOUT mode column (simulate pre-migration data)
	result, err := db.Exec(
		"INSERT INTO topics (book_id, name, chat_file_uuid, created_at, updated_at) VALUES (?, ?, ?, ?, ?)",
		bookID, "Old Topic", "old-uuid", 1000, 1000,
	)
	if err != nil {
		t.Fatalf("Failed to insert old topic: %v", err)
	}

	oldTopicID, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("Failed to get old topic ID: %v", err)
	}

	// Verify old topic gets default mode via migration
	var mode string
	err = db.QueryRow("SELECT mode FROM topics WHERE id = ?", oldTopicID).Scan(&mode)
	if err != nil {
		t.Fatalf("Failed to query mode: %v", err)
	}

	if mode != "scholar" {
		t.Errorf("Expected existing topic to have default mode 'scholar', got %s", mode)
	}
}
