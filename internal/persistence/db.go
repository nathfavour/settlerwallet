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

type UserRules struct {
	TelegramID      int64
	MaxSlippage     float64
	DailyLimit      string // big.Int as string
	CurrentSpend    string // big.Int as string
	LastReset       int64  // Unix timestamp
}

type DB struct {
	conn *sql.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS vaults (
			telegram_id INTEGER PRIMARY KEY,
			encrypted_seed BLOB,
			salt BLOB
		);`,
		`CREATE TABLE IF NOT EXISTS guardrails (
			telegram_id INTEGER PRIMARY KEY,
			max_slippage REAL,
			daily_limit TEXT,
			current_spend TEXT,
			last_reset INTEGER
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return nil, fmt.Errorf("failed to execute migrations: %w", err)
		}
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

func (db *DB) SaveRules(r UserRules) error {
	query := `INSERT OR REPLACE INTO guardrails (telegram_id, max_slippage, daily_limit, current_spend, last_reset) VALUES (?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, r.TelegramID, r.MaxSlippage, r.DailyLimit, r.CurrentSpend, r.LastReset)
	return err
}

func (db *DB) GetRules(telegramID int64) (*UserRules, error) {
	query := `SELECT telegram_id, max_slippage, daily_limit, current_spend, last_reset FROM guardrails WHERE telegram_id = ?`
	row := db.conn.QueryRow(query, telegramID)

	var r UserRules
	if err := row.Scan(&r.TelegramID, &r.MaxSlippage, &r.DailyLimit, &r.CurrentSpend, &r.LastReset); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &r, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
