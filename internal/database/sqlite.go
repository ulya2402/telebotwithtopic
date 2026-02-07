package database

import (
	"database/sql"
	"log"

	_ "modernc.org/sqlite"
)

type DB struct {
	Conn *sql.DB
}

type ChatMessage struct {
	Role    string
	Content string
}

func InitDB(filepath string) *DB {
	db, err := sql.Open("sqlite", filepath)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}

	createPreferencesTable := `
	CREATE TABLE IF NOT EXISTS user_preferences (
		user_id INTEGER PRIMARY KEY,
		language_code TEXT DEFAULT 'en'
	);
	`
	_, err = db.Exec(createPreferencesTable)
	if err != nil {
		log.Fatalf("Error creating preferences table: %v", err)
	}

	createHistoryTable := `
	CREATE TABLE IF NOT EXISTS chat_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		chat_id INTEGER,
		thread_id INTEGER DEFAULT 0,
		role TEXT,
		content TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	_, err = db.Exec(createHistoryTable)
	if err != nil {
		log.Fatalf("Error creating history table: %v", err)
	}

	log.Println("Database and tables initialized successfully")
	return &DB{Conn: db}
}

func (db *DB) SetUserLanguage(userID int64, lang string) error {
	query := `INSERT INTO user_preferences (user_id, language_code) VALUES (?, ?) 
              ON CONFLICT(user_id) DO UPDATE SET language_code = excluded.language_code`
	_, err := db.Conn.Exec(query, userID, lang)
	return err
}

func (db *DB) GetUserLanguage(userID int64) string {
	var lang string
	query := `SELECT language_code FROM user_preferences WHERE user_id = ?`
	err := db.Conn.QueryRow(query, userID).Scan(&lang)
	if err != nil {
		return "en"
	}
	return lang
}

func (db *DB) AddHistory(chatID int64, threadID int, role, content string) error {
	insertQuery := `INSERT INTO chat_history (chat_id, thread_id, role, content) VALUES (?, ?, ?, ?)`
	_, err := db.Conn.Exec(insertQuery, chatID, threadID, role, content)
	if err != nil {
		return err
	}

	pruneQuery := `
		DELETE FROM chat_history 
		WHERE id NOT IN (
			SELECT id FROM chat_history 
			WHERE chat_id = ? AND thread_id = ? 
			ORDER BY id DESC 
			LIMIT 20
		) AND chat_id = ? AND thread_id = ?
	`
	_, err = db.Conn.Exec(pruneQuery, chatID, threadID, chatID, threadID)
	return err
}

func (db *DB) GetHistory(chatID int64, threadID int) ([]ChatMessage, error) {
	query := `SELECT role, content FROM chat_history WHERE chat_id = ? AND thread_id = ? ORDER BY id ASC`
	rows, err := db.Conn.Query(query, chatID, threadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []ChatMessage
	for rows.Next() {
		var msg ChatMessage
		if err := rows.Scan(&msg.Role, &msg.Content); err != nil {
			return nil, err
		}
		history = append(history, msg)
	}
	return history, nil
}