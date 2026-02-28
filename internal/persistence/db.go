package persistence

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite"
)

type AccountType string

const (
	AccountLocal    AccountType = "local"
	AccountTelegram AccountType = "telegram"
)

type Account struct {
	ID         string // "local:<name>" or "tg:<uid>"
	Type       AccountType
	Salt       []byte
	Iterations int
}

type Wallet struct {
	ID            int
	AccountID     string
	Name          string
	Chain         string
	Address       string
	EncryptedSeed []byte
	Salt          []byte
}

type UserRules struct {
	AccountID    string
	MaxSlippage  float64
	DailyLimit   string
	CurrentSpend string
	LastReset    int64
}

type DB struct {
	conn *sql.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			type TEXT,
			salt BLOB,
			iterations INTEGER
		);`,
		`CREATE TABLE IF NOT EXISTS wallets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id TEXT,
			name TEXT,
			chain TEXT,
			address TEXT,
			encrypted_seed BLOB,
			salt BLOB,
			FOREIGN KEY(account_id) REFERENCES accounts(id)
		);`,
		`CREATE TABLE IF NOT EXISTS guardrails (
			account_id TEXT PRIMARY KEY,
			max_slippage REAL,
			daily_limit TEXT,
			current_spend TEXT,
			last_reset INTEGER,
			FOREIGN KEY(account_id) REFERENCES accounts(id)
		);`,
		`CREATE TABLE IF NOT EXISTS config (
			key TEXT PRIMARY KEY,
			value TEXT
		);`,
	}

	for _, query := range queries {
		if _, err := db.Exec(query); err != nil {
			return nil, fmt.Errorf("failed to execute migrations: %w", err)
		}
	}

	return &DB{conn: db}, nil
}

func (db *DB) SaveAccount(a Account) error {
	query := `INSERT OR REPLACE INTO accounts (id, type, salt, iterations) VALUES (?, ?, ?, ?)`
	_, err := db.conn.Exec(query, a.ID, a.Type, a.Salt, a.Iterations)
	return err
}

func (db *DB) GetAccount(id string) (*Account, error) {
	query := `SELECT id, type, salt, iterations FROM accounts WHERE id = ?`
	row := db.conn.QueryRow(query, id)
	var a Account
	if err := row.Scan(&a.ID, &a.Type, &a.Salt, &a.Iterations); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &a, nil
}

func (db *DB) GetAccountsByType(t AccountType) ([]Account, error) {
	query := `SELECT id, type, salt, iterations FROM accounts WHERE type = ?`
	rows, err := db.conn.Query(query, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Type, &a.Salt, &a.Iterations); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (db *DB) SaveWallet(w Wallet) error {
	query := `INSERT INTO wallets (account_id, name, chain, address, encrypted_seed, salt) VALUES (?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, w.AccountID, w.Name, w.Chain, w.Address, w.EncryptedSeed, w.Salt)
	return err
}

func (db *DB) GetWallets(accountID string) ([]Wallet, error) {
	query := `SELECT id, account_id, name, chain, address, encrypted_seed, salt FROM wallets WHERE account_id = ?`
	rows, err := db.conn.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []Wallet
	for rows.Next() {
		var w Wallet
		if err := rows.Scan(&w.ID, &w.AccountID, &w.Name, &w.Chain, &w.Address, &w.EncryptedSeed, &w.Salt); err != nil {
			return nil, err
		}
		wallets = append(wallets, w)
	}
	return wallets, nil
}

func (db *DB) SaveRules(r UserRules) error {
	query := `INSERT OR REPLACE INTO guardrails (account_id, max_slippage, daily_limit, current_spend, last_reset) VALUES (?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, r.AccountID, r.MaxSlippage, r.DailyLimit, r.CurrentSpend, r.LastReset)
	return err
}

func (db *DB) GetRules(accountID string) (*UserRules, error) {
	query := `SELECT account_id, max_slippage, daily_limit, current_spend, last_reset FROM guardrails WHERE account_id = ?`
	row := db.conn.QueryRow(query, accountID)
	var r UserRules
	if err := row.Scan(&r.AccountID, &r.MaxSlippage, &r.DailyLimit, &r.CurrentSpend, &r.LastReset); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &r, nil
}

func (db *DB) SetConfig(key, value string) error {
	query := `INSERT OR REPLACE INTO config (key, value) VALUES (?, ?)`
	_, err := db.conn.Exec(query, key, value)
	return err
}

func (db *DB) GetConfig(key string) (string, error) {
	query := `SELECT value FROM config WHERE key = ?`
	row := db.conn.QueryRow(query, key)
	var value string
	if err := row.Scan(&value); err != nil {
		if err == sql.ErrNoRows {
			return "", nil
		}
		return "", err
	}
	return value, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}
