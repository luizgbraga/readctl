package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

func GetDocumentsDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}

	documentsDir := filepath.Join(dataDir, "documents")

	if err := os.MkdirAll(documentsDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create documents directory: %w", err)
	}

	return documentsDir, nil
}

func SaveDocument(uuid, content string) error {
	documentsDir, err := GetDocumentsDir()
	if err != nil {
		return err
	}

	filePath := filepath.Join(documentsDir, fmt.Sprintf("%s.md", uuid))

	if err := os.WriteFile(filePath, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write document file: %w", err)
	}

	return nil
}

// Returns empty string (not error) if document doesn't exist
func LoadDocument(uuid string) (string, error) {
	documentsDir, err := GetDocumentsDir()
	if err != nil {
		return "", err
	}

	filePath := filepath.Join(documentsDir, fmt.Sprintf("%s.md", uuid))

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Document doesn't exist yet - return empty string
			return "", nil
		}
		return "", fmt.Errorf("failed to read document file: %w", err)
	}

	return string(data), nil
}

func DocumentExists(uuid string) bool {
	documentsDir, err := GetDocumentsDir()
	if err != nil {
		return false
	}

	filePath := filepath.Join(documentsDir, fmt.Sprintf("%s.md", uuid))

	_, err = os.Stat(filePath)
	return err == nil
}
