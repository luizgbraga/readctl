package storage

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Topic struct {
	ID           int
	BookID       int
	Name         string
	ChatFileUUID string
	Mode         string
	CreatedAt    int64
	UpdatedAt    int64
}

func CreateTopic(db *sql.DB, bookID int, name string, mode string) (int, string, error) {
	now := time.Now().Unix()
	chatUUID := uuid.New().String()

	// Default to scholar mode if empty or whitespace
	if len(mode) == 0 || len(mode) > 0 && mode[0] == ' ' {
		mode = "scholar"
	}
	trimmedMode := mode
	if len(mode) > 0 {
		// Simple trim for spaces
		start := 0
		end := len(mode)
		for start < end && mode[start] == ' ' {
			start++
		}
		for end > start && mode[end-1] == ' ' {
			end--
		}
		trimmedMode = mode[start:end]
		if trimmedMode == "" {
			trimmedMode = "scholar"
		}
	} else {
		trimmedMode = "scholar"
	}

	result, err := db.Exec(
		"INSERT INTO topics (book_id, name, chat_file_uuid, mode, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)",
		bookID, name, chatUUID, trimmedMode, now, now,
	)
	if err != nil {
		return 0, "", fmt.Errorf("failed to create topic: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, "", fmt.Errorf("failed to get topic ID: %w", err)
	}

	return int(id), chatUUID, nil
}

func GetTopics(db *sql.DB, bookID int) ([]Topic, error) {
	rows, err := db.Query(
		"SELECT id, book_id, name, chat_file_uuid, mode, created_at, updated_at FROM topics WHERE book_id = ? ORDER BY created_at DESC",
		bookID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query topics: %w", err)
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		var t Topic
		if err := rows.Scan(&t.ID, &t.BookID, &t.Name, &t.ChatFileUUID, &t.Mode, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan topic: %w", err)
		}
		topics = append(topics, t)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating topics: %w", err)
	}

	return topics, nil
}

func DeleteTopic(db *sql.DB, id int) error {
	// Get topic to find chat file UUID
	var chatUUID string
	err := db.QueryRow("SELECT chat_file_uuid FROM topics WHERE id = ?", id).Scan(&chatUUID)
	if err != nil {
		return fmt.Errorf("failed to get topic chat UUID: %w", err)
	}

	// Delete chat file
	if err := DeleteChatFile(chatUUID); err != nil {
		// Log error but continue - file might not exist
		// In production, would use proper logging
	}

	// Delete topic from database
	_, err = db.Exec("DELETE FROM topics WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete topic: %w", err)
	}

	return nil
}

func UpdateTopicMode(db *sql.DB, topicID int, mode string) error {
	now := time.Now().Unix()
	_, err := db.Exec(
		"UPDATE topics SET mode = ?, updated_at = ? WHERE id = ?",
		mode, now, topicID,
	)
	if err != nil {
		return fmt.Errorf("failed to update topic mode: %w", err)
	}
	return nil
}
