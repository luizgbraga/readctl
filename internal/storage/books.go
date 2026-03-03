package storage

import (
	"database/sql"
	"fmt"
	"time"
)

type Book struct {
	ID           int
	Title        string
	Author       string
	LastAccessed int64
	CreatedAt    int64
}

func CreateBook(db *sql.DB, title, author string) (int, error) {
	now := time.Now().Unix()

	result, err := db.Exec(
		"INSERT INTO books (title, author, last_accessed, created_at) VALUES (?, ?, ?, ?)",
		title, author, now, now,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to create book: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("failed to get book ID: %w", err)
	}

	return int(id), nil
}

func GetBooks(db *sql.DB) ([]Book, error) {
	rows, err := db.Query(
		"SELECT id, title, author, last_accessed, created_at FROM books ORDER BY last_accessed DESC",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query books: %w", err)
	}
	defer rows.Close()

	var books []Book
	for rows.Next() {
		var b Book
		if err := rows.Scan(&b.ID, &b.Title, &b.Author, &b.LastAccessed, &b.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan book: %w", err)
		}
		books = append(books, b)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating books: %w", err)
	}

	return books, nil
}

func DeleteBook(db *sql.DB, id int) error {
	// Get all topics for this book to delete their chat files
	topics, err := GetTopics(db, id)
	if err != nil {
		return fmt.Errorf("failed to get topics for book deletion: %w", err)
	}

	// Delete chat files for all topics
	for _, topic := range topics {
		if err := DeleteChatFile(topic.ChatFileUUID); err != nil {
			// Log error but continue - file might not exist
			// In production, would use proper logging
		}
	}

	// Delete book (CASCADE will delete topics)
	_, err = db.Exec("DELETE FROM books WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("failed to delete book: %w", err)
	}

	return nil
}

func UpdateLastAccessed(db *sql.DB, id int) error {
	now := time.Now().Unix()

	_, err := db.Exec("UPDATE books SET last_accessed = ? WHERE id = ?", now, id)
	if err != nil {
		return fmt.Errorf("failed to update last accessed: %w", err)
	}

	return nil
}
