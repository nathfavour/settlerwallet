package blockchain

import (
	"context"
	"math/big"

	"github.com/nathfavour/settlerwallet/internal/vault"
)

// Balance represents a cross-chain balance.
type Balance struct {
	Chain    vault.Chain
	Address  string
	Symbol   string
	Amount   *big.Int
	Decimals int
}

// Transfer represents a cross-chain transfer request.
type Transfer struct {
	To     string
	Amount *big.Int
	Token  string // Empty for native token
}

// TransactionResult contains the outcome of a broadcasted transaction.
type TransactionResult struct {
	Hash    string
	Success bool
	Error   error
}

// Client defines the interface for interacting with different blockchains.
type Client interface {
	GetBalance(ctx context.Context, address string) (*Balance, error)
	GetTokenBalances(ctx context.Context, address string) ([]*Balance, error)
	Transfer(ctx context.Context, from *vault.DerivedKey, req Transfer) (*TransactionResult, error)
}

