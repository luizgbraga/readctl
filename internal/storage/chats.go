package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type Message struct {
	Role      string `json:"role"`      // "user" or "assistant"
	Content   string `json:"content"`   // Message text
	Timestamp int64  `json:"timestamp"` // Unix timestamp
}

func SaveChat(uuid string, messages []Message) error {
	chatsDir, err := GetChatsDir()
	if err != nil {
		return fmt.Errorf("failed to get chats directory: %w", err)
	}

	filePath := filepath.Join(chatsDir, fmt.Sprintf("%s.json", uuid))

	data, err := json.MarshalIndent(messages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal messages: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write chat file: %w", err)
	}

	return nil
}

func LoadChat(uuid string) ([]Message, error) {
	chatsDir, err := GetChatsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get chats directory: %w", err)
	}

	filePath := filepath.Join(chatsDir, fmt.Sprintf("%s.json", uuid))

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Chat file doesn't exist yet - return empty messages
			return []Message{}, nil
		}
		return nil, fmt.Errorf("failed to read chat file: %w", err)
	}

	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal messages: %w", err)
	}

	return messages, nil
}

func DeleteChatFile(uuid string) error {
	chatsDir, err := GetChatsDir()
	if err != nil {
		return fmt.Errorf("failed to get chats directory: %w", err)
	}

	filePath := filepath.Join(chatsDir, fmt.Sprintf("%s.json", uuid))

	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete chat file: %w", err)
	}

	return nil
}

// Archive files use the pattern: {uuid}_archive.json
func SaveChatArchive(uuid string, messages []Message) error {
	chatsDir, err := GetChatsDir()
	if err != nil {
		return fmt.Errorf("failed to get chats directory: %w", err)
	}

	archivePath := filepath.Join(chatsDir, fmt.Sprintf("%s_archive.json", uuid))

	// Load existing archive if it exists
	var existingMessages []Message
	if data, err := os.ReadFile(archivePath); err == nil {
		// File exists, load it
		if err := json.Unmarshal(data, &existingMessages); err != nil {
			return fmt.Errorf("failed to unmarshal existing archive: %w", err)
		}
	}

	// Append new messages to existing
	combinedMessages := append(existingMessages, messages...)

	// Marshal with indentation for readability
	data, err := json.MarshalIndent(combinedMessages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal archive messages: %w", err)
	}

	// Write with same permissions as main chat files
	if err := os.WriteFile(archivePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write archive file: %w", err)
	}

	return nil
}

// Returns empty slice (not error) if archive doesn't exist
func LoadChatArchive(uuid string) ([]Message, error) {
	chatsDir, err := GetChatsDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get chats directory: %w", err)
	}

	archivePath := filepath.Join(chatsDir, fmt.Sprintf("%s_archive.json", uuid))

	data, err := os.ReadFile(archivePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Archive doesn't exist - return empty slice, not error
			return []Message{}, nil
		}
		return nil, fmt.Errorf("failed to read archive file: %w", err)
	}

	var messages []Message
	if err := json.Unmarshal(data, &messages); err != nil {
		return nil, fmt.Errorf("failed to unmarshal archive messages: %w", err)
	}

	return messages, nil
}
