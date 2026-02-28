package blockchain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/nathfavour/settlerwallet/internal/vault"
)

type SolanaClient struct {
	client *rpc.Client
}

func NewSolanaClient(rpcURL string) (*SolanaClient, error) {
	client := rpc.New(rpcURL)
	return &SolanaClient{client: client}, nil
}

func (c *SolanaClient) GetBalance(ctx context.Context, address string) (*Balance, error) {
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, err
	}

	balance, err := c.client.GetBalance(ctx, pubKey, rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}

	return &Balance{
		Chain:    vault.ChainSolana,
		Address:  address,
		Symbol:   "SOL",
		Amount:   big.NewInt(int64(balance.Value)),
		Decimals: 9,
	}, nil
}

func (c *SolanaClient) Transfer(ctx context.Context, from *vault.DerivedKey, req Transfer) (*TransactionResult, error) {
	if from.Chain != vault.ChainSolana {
		return nil, fmt.Errorf("invalid chain for Solana transfer")
	}


	fromPubKey, err := solana.PublicKeyFromBase58(from.Address)
	if err != nil {
		return nil, err
	}

	toPubKey, err := solana.PublicKeyFromBase58(req.To)
	if err != nil {
		return nil, err
	}

	recent, err := c.client.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}

	tx, err := solana.NewTransaction(
		[]solana.Instruction{
			system.NewTransferInstruction(
				req.Amount.Uint64(),
				fromPubKey,
				toPubKey,
			).Build(),
		},
		recent.Value.Blockhash,
		solana.TransactionPayer(fromPubKey),
	)
	if err != nil {
		return nil, err
	}

	privKey := solana.PrivateKey(from.PrivateKey)
	_, err = tx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		if key.Equals(fromPubKey) {
			return &privKey
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sig, err := c.client.SendTransaction(ctx, tx)
	if err != nil {
		return nil, err
	}

	return &TransactionResult{
		Hash:    sig.String(),
		Success: true,
	}, nil
}
