package storage

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	_ "github.com/mattn/go-sqlite3"
)

func InitDB() (*sql.DB, error) {
	// Create data directory using XDG conventions
	dataPath, err := xdg.DataFile("readctl/readctl.db")
	if err != nil {
		return nil, fmt.Errorf("failed to get data path: %w", err)
	}

	// Ensure the directory exists
	if err := os.MkdirAll(filepath.Dir(dataPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	// Create chats directory
	chatsDir := filepath.Join(filepath.Dir(dataPath), "chats")
	if err := os.MkdirAll(chatsDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create chats directory: %w", err)
	}

	// Open database with WAL mode for better concurrency
	db, err := sql.Open("sqlite3", fmt.Sprintf("file:%s?_journal_mode=WAL&_busy_timeout=5000", dataPath))
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Initialize schema
	if err := initSchema(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

func initSchema(db *sql.DB) error {
	schema := `
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER PRIMARY KEY
	);

	CREATE TABLE IF NOT EXISTS books (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title TEXT NOT NULL,
		author TEXT NOT NULL,
		last_accessed INTEGER DEFAULT 0,
		created_at INTEGER
	);

	CREATE TABLE IF NOT EXISTS topics (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		book_id INTEGER NOT NULL,
		name TEXT NOT NULL,
		chat_file_uuid TEXT NOT NULL,
		created_at INTEGER,
		updated_at INTEGER,
		FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_topics_book_id ON topics(book_id);
	CREATE INDEX IF NOT EXISTS idx_books_last_accessed ON books(last_accessed DESC);
	CREATE INDEX IF NOT EXISTS idx_topics_created ON topics(created_at DESC);
	`

	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	// Insert schema version if not exists
	var version int
	err := db.QueryRow("SELECT version FROM schema_version WHERE version = 1").Scan(&version)
	if err == sql.ErrNoRows {
		if _, err := db.Exec("INSERT INTO schema_version (version) VALUES (1)"); err != nil {
			return fmt.Errorf("failed to insert schema version: %w", err)
		}
	}

	// Add mode column to topics table if it doesn't exist (idempotent migration)
	var modeColumnExists int
	err = db.QueryRow("SELECT COUNT(*) FROM pragma_table_info('topics') WHERE name = 'mode'").Scan(&modeColumnExists)
	if err != nil {
		return fmt.Errorf("failed to check for mode column: %w", err)
	}

	if modeColumnExists == 0 {
		if _, err := db.Exec("ALTER TABLE topics ADD COLUMN mode TEXT DEFAULT 'scholar'"); err != nil {
			return fmt.Errorf("failed to add mode column: %w", err)
		}
	}

	return nil
}

func GetDataDir() (string, error) {
	dataPath, err := xdg.DataFile("readctl/readctl.db")
	if err != nil {
		return "", fmt.Errorf("failed to get data path: %w", err)
	}
	return filepath.Dir(dataPath), nil
}

func GetChatsDir() (string, error) {
	dataDir, err := GetDataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dataDir, "chats"), nil
}
