package guardrail

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/nathfavour/settlerwallet/internal/blockchain"
)

var (
	ErrSlippageTooHigh = errors.New("slippage exceeds maximum allowed")
	ErrDailySpendLimit = errors.New("daily spend limit exceeded")
	ErrBlockedContract = errors.New("interaction with blocked contract")
)

// Rules define the user-specific safety thresholds.
type Rules struct {
	MaxSlippage      float64
	DailySpendLimit  *big.Int
	CurrentDailySpend *big.Int
	WhitelistedKeys  []string
	BlockedKeys      []string
}

// Engine intercepts and validates transaction proposals.
type Engine struct {
	mu    sync.RWMutex
	rules map[string]*Rules // UserID -> Rules
}

func NewEngine() *Engine {
	return &Engine{
		rules: make(map[string]*Rules),
	}
}

// ValidateProposal checks if a transfer proposal meets the safety rules.
func (e *Engine) ValidateProposal(ctx context.Context, userID string, req blockchain.Transfer) error {
	e.mu.RLock()
	rules, exists := e.rules[userID]
	e.mu.RUnlock()

	if !exists {
		// Use default rules if none specified for the user.
		rules = &Rules{
			MaxSlippage:       1.0,
			DailySpendLimit:   big.NewInt(1000000000000000000), // 1 ETH/BNB
			CurrentDailySpend: big.NewInt(0),
		}
	}

	// 1. Check daily spend limit
	newSpend := new(big.Int).Add(rules.CurrentDailySpend, req.Amount)
	if newSpend.Cmp(rules.DailySpendLimit) > 0 {
		return ErrDailySpendLimit
	}

	// 2. Check for blocked contracts (simplified here)
	for _, blocked := range rules.BlockedKeys {
		if req.To == blocked {
			return ErrBlockedContract
		}
	}

	return nil
}

// SetRules updates the safety rules for a user.
func (e *Engine) SetRules(userID string, rules *Rules) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.rules[userID] = rules
}
