package guardrail

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/nathfavour/settlerwallet/internal/blockchain"
	"github.com/nathfavour/settlerwallet/internal/persistence"
)

var (
	ErrSlippageTooHigh = errors.New("slippage exceeds maximum allowed")
	ErrDailySpendLimit = errors.New("daily spend limit exceeded")
	ErrBlockedContract = errors.New("interaction with blocked contract")
)

// Engine intercepts and validates transaction proposals.
type Engine struct {
	mu sync.Mutex
	db *persistence.DB
}

func NewEngine(db *persistence.DB) *Engine {
	return &Engine{
		db: db,
	}
}

// ValidateProposal checks if a transfer proposal meets the safety rules.
func (e *Engine) ValidateProposal(ctx context.Context, userID int64, req blockchain.Transfer) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, err := e.db.GetRules(userID)
	if err != nil {
		return err
	}

	if rules == nil {
		// Default rules: 1 unit (BNB/SOL) per day
		rules = &persistence.UserRules{
			TelegramID:   userID,
			MaxSlippage:  1.0,
			DailyLimit:   "1000000000000000000", // Defaulting to 1e18 for simplicity
			CurrentSpend: "0",
			LastReset:    time.Now().Unix(),
		}
	}

	// Reset daily spend if 24h have passed
	now := time.Now().Unix()
	if now-rules.LastReset >= 86400 {
		rules.CurrentSpend = "0"
		rules.LastReset = now
	}

	limit, _ := new(big.Int).SetString(rules.DailyLimit, 10)
	current, _ := new(big.Int).SetString(rules.CurrentSpend, 10)

	newSpend := new(big.Int).Add(current, req.Amount)
	if newSpend.Cmp(limit) > 0 {
		return ErrDailySpendLimit
	}

	// If valid, update and save (this is just validation, actual increment happens on success)
	// rules.CurrentSpend = newSpend.String()
	// e.db.SaveRules(*rules)

	return nil
}

// RecordSpend persists the incremented spend after a successful transaction.
func (e *Engine) RecordSpend(userID int64, amount *big.Int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, err := e.db.GetRules(userID)
	if err != nil || rules == nil {
		return err
	}

	current, _ := new(big.Int).SetString(rules.CurrentSpend, 10)
	newSpend := new(big.Int).Add(current, amount)
	rules.CurrentSpend = newSpend.String()

	return e.db.SaveRules(*rules)
}

// SetLimit updates the daily spend limit for a user.
func (e *Engine) SetLimit(userID int64, limit *big.Int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, err := e.db.GetRules(userID)
	if err != nil {
		return err
	}

	if rules == nil {
		rules = &persistence.UserRules{
			TelegramID:   userID,
			MaxSlippage:  1.0,
			CurrentSpend: "0",
			LastReset:    time.Now().Unix(),
		}
	}

	rules.DailyLimit = limit.String()
	return e.db.SaveRules(*rules)
}
