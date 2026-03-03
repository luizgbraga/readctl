package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Test helper to get unique test UUID and ensure clean slate
func setupTestArchive(t *testing.T) (string, string) {
	// Get the actual chats directory
	chatsDir, err := GetChatsDir()
	if err != nil {
		t.Fatalf("Failed to get chats dir: %v", err)
	}

	// Use test name to create unique UUID
	uuid := "test-" + t.Name()

	// Clean up any existing archive file from previous test runs
	archivePath := filepath.Join(chatsDir, uuid+"_archive.json")
	os.Remove(archivePath) // Ignore errors - file may not exist

	// Register cleanup
	t.Cleanup(func() {
		os.Remove(archivePath)
	})

	return uuid, chatsDir
}

func TestSaveChatArchive(t *testing.T) {
	uuid, chatsDir := setupTestArchive(t)

	messages := []Message{
		{Role: "user", Content: "First message", Timestamp: time.Now().Unix()},
		{Role: "assistant", Content: "First response", Timestamp: time.Now().Unix()},
	}

	// Test: Save archive creates file
	err := SaveChatArchive(uuid, messages)
	if err != nil {
		t.Fatalf("SaveChatArchive failed: %v", err)
	}

	archivePath := filepath.Join(chatsDir, uuid+"_archive.json")
	if _, err := os.Stat(archivePath); os.IsNotExist(err) {
		t.Errorf("Archive file was not created at %s", archivePath)
	}

	// Test: Load archive returns saved messages
	loaded, err := LoadChatArchive(uuid)
	if err != nil {
		t.Fatalf("LoadChatArchive failed: %v", err)
	}

	if len(loaded) != len(messages) {
		t.Errorf("Expected %d messages, got %d", len(messages), len(loaded))
	}

	for i := range messages {
		if loaded[i].Role != messages[i].Role {
			t.Errorf("Message %d: expected role %s, got %s", i, messages[i].Role, loaded[i].Role)
		}
		if loaded[i].Content != messages[i].Content {
			t.Errorf("Message %d: expected content %s, got %s", i, messages[i].Content, loaded[i].Content)
		}
	}
}

func TestSaveChatArchiveAppend(t *testing.T) {
	uuid, _ := setupTestArchive(t)

	// First save
	firstMessages := []Message{
		{Role: "user", Content: "First batch", Timestamp: 1000},
	}
	err := SaveChatArchive(uuid, firstMessages)
	if err != nil {
		t.Fatalf("First SaveChatArchive failed: %v", err)
	}

	// Second save should append
	secondMessages := []Message{
		{Role: "user", Content: "Second batch", Timestamp: 2000},
	}
	err = SaveChatArchive(uuid, secondMessages)
	if err != nil {
		t.Fatalf("Second SaveChatArchive failed: %v", err)
	}

	// Load should return both batches
	loaded, err := LoadChatArchive(uuid)
	if err != nil {
		t.Fatalf("LoadChatArchive failed: %v", err)
	}

	expectedCount := len(firstMessages) + len(secondMessages)
	if len(loaded) != expectedCount {
		t.Errorf("Expected %d messages after append, got %d", expectedCount, len(loaded))
	}

	// Verify order: first batch then second batch
	if loaded[0].Content != "First batch" {
		t.Errorf("Expected first message to be 'First batch', got %s", loaded[0].Content)
	}
	if loaded[1].Content != "Second batch" {
		t.Errorf("Expected second message to be 'Second batch', got %s", loaded[1].Content)
	}
}

func TestLoadChatArchiveNonExistent(t *testing.T) {
	uuid, _ := setupTestArchive(t)

	// Don't create archive, just try to load non-existent one

	// Test: Loading non-existent archive returns empty slice, not error
	loaded, err := LoadChatArchive(uuid)
	if err != nil {
		t.Fatalf("LoadChatArchive should not error on non-existent file, got: %v", err)
	}

	if loaded == nil {
		t.Error("LoadChatArchive should return empty slice, not nil")
	}

	if len(loaded) != 0 {
		t.Errorf("LoadChatArchive should return empty slice, got %d messages", len(loaded))
	}
}
