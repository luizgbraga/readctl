package storage

import (
	"os"
	"path/filepath"
	"testing"
)

// Test helper to set up clean test environment
func setupTestDocuments(t *testing.T) (string, string) {
	// Get the actual documents directory
	documentsDir, err := GetDocumentsDir()
	if err != nil {
		t.Fatalf("Failed to get documents dir: %v", err)
	}

	// Use test name to create unique UUID
	uuid := "test-doc-" + t.Name()

	// Clean up any existing document file from previous test runs
	docPath := filepath.Join(documentsDir, uuid+".md")
	os.Remove(docPath) // Ignore errors - file may not exist

	// Register cleanup
	t.Cleanup(func() {
		os.Remove(docPath)
	})

	return uuid, documentsDir
}

func TestGetDocumentsDir(t *testing.T) {
	dir, err := GetDocumentsDir()
	if err != nil {
		t.Fatalf("GetDocumentsDir failed: %v", err)
	}

	// Verify directory was created
	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("Documents directory does not exist: %v", err)
	}

	if !info.IsDir() {
		t.Error("GetDocumentsDir returned a path that is not a directory")
	}

	// Verify it's under the data directory
	dataDir, err := GetDataDir()
	if err != nil {
		t.Fatalf("Failed to get data dir: %v", err)
	}

	expectedDir := filepath.Join(dataDir, "documents")
	if dir != expectedDir {
		t.Errorf("Expected documents dir %s, got %s", expectedDir, dir)
	}
}

func TestSaveLoadDocument(t *testing.T) {
	uuid, documentsDir := setupTestDocuments(t)

	content := "# Test Document\n\nThis is a test document with **markdown** formatting.\n\n- Point 1\n- Point 2\n"

	// Test: SaveDocument creates file
	err := SaveDocument(uuid, content)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	// Verify file was created
	docPath := filepath.Join(documentsDir, uuid+".md")
	if _, err := os.Stat(docPath); os.IsNotExist(err) {
		t.Errorf("Document file was not created at %s", docPath)
	}

	// Verify file permissions (should be 0600 for privacy)
	info, err := os.Stat(docPath)
	if err != nil {
		t.Fatalf("Failed to stat document file: %v", err)
	}
	mode := info.Mode().Perm()
	if mode != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", mode)
	}

	// Test: LoadDocument retrieves content
	loaded, err := LoadDocument(uuid)
	if err != nil {
		t.Fatalf("LoadDocument failed: %v", err)
	}

	if loaded != content {
		t.Errorf("LoadDocument returned different content.\nExpected:\n%s\nGot:\n%s", content, loaded)
	}
}

func TestLoadNonexistentDocument(t *testing.T) {
	uuid, _ := setupTestDocuments(t)

	// Don't create document, just try to load non-existent one

	// Test: Loading non-existent document returns empty string and nil error
	loaded, err := LoadDocument(uuid)
	if err != nil {
		t.Fatalf("LoadDocument should not error on non-existent file, got: %v", err)
	}

	if loaded != "" {
		t.Errorf("LoadDocument should return empty string for non-existent document, got: %s", loaded)
	}
}

func TestDocumentExists(t *testing.T) {
	uuid, _ := setupTestDocuments(t)

	// Test: DocumentExists returns false for non-existent document
	exists := DocumentExists(uuid)
	if exists {
		t.Error("DocumentExists should return false for non-existent document")
	}

	// Create document
	content := "# Test\nContent here"
	err := SaveDocument(uuid, content)
	if err != nil {
		t.Fatalf("SaveDocument failed: %v", err)
	}

	// Test: DocumentExists returns true for existing document
	exists = DocumentExists(uuid)
	if !exists {
		t.Error("DocumentExists should return true for existing document")
	}
}
