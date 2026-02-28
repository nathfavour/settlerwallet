package persistence

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

type UserVault struct {
	TelegramID    int64
	EncryptedSeed []byte
	Salt          []byte
}

type DB struct {
	conn *sql.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	query := `
	CREATE TABLE IF NOT EXISTS vaults (
		telegram_id INTEGER PRIMARY KEY,
		encrypted_seed BLOB,
		salt BLOB
	);`

	if _, err := db.Exec(query); err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &DB{conn: db}, nil
}

func (db *DB) SaveVault(v UserVault) error {
	query := `INSERT OR REPLACE INTO vaults (telegram_id, encrypted_seed, salt) VALUES (?, ?, ?)`
	_, err := db.conn.Exec(query, v.TelegramID, v.EncryptedSeed, v.Salt)
	return err
}

func (db *DB) GetVault(telegramID int64) (*UserVault, error) {
	query := `SELECT telegram_id, encrypted_seed, salt FROM vaults WHERE telegram_id = ?`
	row := db.conn.QueryRow(query, telegramID)

	var v UserVault
	if err := row.Scan(&v.TelegramID, &v.EncryptedSeed, &v.Salt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &v, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
