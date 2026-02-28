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

type Engine struct {
	mu sync.Mutex
	db *persistence.DB
}

func NewEngine(db *persistence.DB) *Engine {
	return &Engine{
		db: db,
	}
}

func (e *Engine) ValidateProposal(ctx context.Context, accountID string, req blockchain.Transfer) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, err := e.db.GetRules(accountID)
	if err != nil {
		return err
	}

	if rules == nil {
		rules = &persistence.UserRules{
			AccountID:    accountID,
			MaxSlippage:  1.0,
			DailyLimit:   "1000000000000000000",
			CurrentSpend: "0",
			LastReset:    time.Now().Unix(),
		}
	}

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

	return nil
}

func (e *Engine) RecordSpend(accountID string, amount *big.Int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, err := e.db.GetRules(accountID)
	if err != nil || rules == nil {
		return err
	}

	current, _ := new(big.Int).SetString(rules.CurrentSpend, 10)
	newSpend := new(big.Int).Add(current, amount)
	rules.CurrentSpend = newSpend.String()

	return e.db.SaveRules(*rules)
}

func (e *Engine) SetLimit(accountID string, limit *big.Int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rules, err := e.db.GetRules(accountID)
	if err != nil {
		return err
	}

	if rules == nil {
		rules = &persistence.UserRules{
			AccountID:    accountID,
			MaxSlippage:  1.0,
			CurrentSpend: "0",
			LastReset:    time.Now().Unix(),
		}
	}

	rules.DailyLimit = limit.String()
	return e.db.SaveRules(*rules)
}
