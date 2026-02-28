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
	LinkedTGID int64
}

type LinkRequest struct {
	AccountID string
	TGID      int64
	Code      string
	ExpiresAt int64
}

type Wallet struct {
	ID                int
	AccountID         string
	Name              string
	Chain             string
	Address           string
	EncryptedSeed     []byte
	EncryptedMnemonic []byte
	Salt              []byte
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

	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	queries := []string{
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			type TEXT,
			salt BLOB,
			iterations INTEGER,
			linked_tg_id INTEGER UNIQUE
		);`,
		`CREATE TABLE IF NOT EXISTS link_requests (
			account_id TEXT PRIMARY KEY,
			tg_id INTEGER,
			code TEXT,
			expires_at INTEGER,
			FOREIGN KEY(account_id) REFERENCES accounts(id) ON UPDATE CASCADE ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS wallets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			account_id TEXT,
			name TEXT,
			chain TEXT,
			address TEXT,
			encrypted_seed BLOB,
			encrypted_mnemonic BLOB,
			salt BLOB,
			FOREIGN KEY(account_id) REFERENCES accounts(id) ON UPDATE CASCADE ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS guardrails (
			account_id TEXT PRIMARY KEY,
			max_slippage REAL,
			daily_limit TEXT,
			current_spend TEXT,
			last_reset INTEGER,
			FOREIGN KEY(account_id) REFERENCES accounts(id) ON UPDATE CASCADE ON DELETE CASCADE
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
	var linkedID interface{}
	if a.LinkedTGID != 0 {
		linkedID = a.LinkedTGID
	}
	query := `INSERT OR REPLACE INTO accounts (id, type, salt, iterations, linked_tg_id) VALUES (?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, a.ID, a.Type, a.Salt, a.Iterations, linkedID)
	return err
}

func (db *DB) RenameAccount(oldID, newID string) error {
	query := `UPDATE accounts SET id = ? WHERE id = ?`
	_, err := db.conn.Exec(query, newID, oldID)
	return err
}

func (db *DB) GetAccount(id string) (*Account, error) {
	query := `SELECT id, type, salt, iterations, linked_tg_id FROM accounts WHERE id = ?`
	row := db.conn.QueryRow(query, id)
	var a Account
	var linkedID sql.NullInt64
	if err := row.Scan(&a.ID, &a.Type, &a.Salt, &a.Iterations, &linkedID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if linkedID.Valid {
		a.LinkedTGID = linkedID.Int64
	}
	return &a, nil
}

func (db *DB) GetAccountByLinkedTGID(tgID int64) (*Account, error) {
	query := `SELECT id, type, salt, iterations, linked_tg_id FROM accounts WHERE linked_tg_id = ?`
	row := db.conn.QueryRow(query, tgID)
	var a Account
	var linkedID sql.NullInt64
	if err := row.Scan(&a.ID, &a.Type, &a.Salt, &a.Iterations, &linkedID); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	if linkedID.Valid {
		a.LinkedTGID = linkedID.Int64
	}
	return &a, nil
}

func (db *DB) CreateLinkRequest(lr LinkRequest) error {
	query := `INSERT OR REPLACE INTO link_requests (account_id, tg_id, code, expires_at) VALUES (?, ?, ?, ?)`
	_, err := db.conn.Exec(query, lr.AccountID, lr.TGID, lr.Code, lr.ExpiresAt)
	return err
}

func (db *DB) GetLinkRequestByTGID(tgID int64) (*LinkRequest, error) {
	query := `SELECT account_id, tg_id, code, expires_at FROM link_requests WHERE tg_id = ?`
	row := db.conn.QueryRow(query, tgID)
	var lr LinkRequest
	if err := row.Scan(&lr.AccountID, &lr.TGID, &lr.Code, &lr.ExpiresAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return &lr, nil
}

func (db *DB) DeleteLinkRequest(accountID string) error {
	query := `DELETE FROM link_requests WHERE account_id = ?`
	_, err := db.conn.Exec(query, accountID)
	return err
}

func (db *DB) GetAccountsByType(t AccountType) ([]Account, error) {
	query := `SELECT id, type, salt, iterations, linked_tg_id FROM accounts WHERE type = ?`
	rows, err := db.conn.Query(query, t)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var linkedID sql.NullInt64
		if err := rows.Scan(&a.ID, &a.Type, &a.Salt, &a.Iterations, &linkedID); err != nil {
			return nil, err
		}
		if linkedID.Valid {
			a.LinkedTGID = linkedID.Int64
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

func (db *DB) SaveWallet(w Wallet) error {
	query := `INSERT INTO wallets (account_id, name, chain, address, encrypted_seed, encrypted_mnemonic, salt) VALUES (?, ?, ?, ?, ?, ?, ?)`
	_, err := db.conn.Exec(query, w.AccountID, w.Name, w.Chain, w.Address, w.EncryptedSeed, w.EncryptedMnemonic, w.Salt)
	return err
}

func (db *DB) GetWallets(accountID string) ([]Wallet, error) {
	query := `SELECT id, account_id, name, chain, address, encrypted_seed, encrypted_mnemonic, salt FROM wallets WHERE account_id = ?`
	rows, err := db.conn.Query(query, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var wallets []Wallet
	for rows.Next() {
		var w Wallet
		if err := rows.Scan(&w.ID, &w.AccountID, &w.Name, &w.Chain, &w.Address, &w.EncryptedSeed, &w.EncryptedMnemonic, &w.Salt); err != nil {
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

func (db *DB) SetUserConfig(tgID int64, key, value string) error {
	fullKey := fmt.Sprintf("user:%d:%s", tgID, key)
	return db.SetConfig(fullKey, value)
}

func (db *DB) GetUserConfig(tgID int64, key string) (string, error) {
	fullKey := fmt.Sprintf("user:%d:%s", tgID, key)
	return db.GetConfig(fullKey)
}

func (db *DB) Close() error {
	return db.conn.Close()
}
