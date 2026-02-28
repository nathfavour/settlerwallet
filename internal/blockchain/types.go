package blockchain

import (
	"context"
	"math/big"
)

// Chain defines the supported networks.
type Chain string

const (
	BNB    Chain = "BNB"
	Solana Chain = "Solana"
)

// Balance represents a cross-chain balance.
type Balance struct {
	Chain    Chain
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
	Transfer(ctx context.Context, from *DerivedKey, req Transfer) (*TransactionResult, error)
}

// DerivedKey placeholder to avoid circular dependency if needed, 
// but we can import vault if we structure carefully.
type DerivedKey struct {
	PrivateKey []byte
	Address    string
	Chain      Chain
}
