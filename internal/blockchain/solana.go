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

func (c *SolanaClient) GetTokenBalances(ctx context.Context, address string) ([]*Balance, error) {
	pubKey, err := solana.PublicKeyFromBase58(address)
	if err != nil {
		return nil, err
	}

	// Filter for all token accounts (SPL Token Program)
	out, err := c.client.GetTokenAccountsByOwner(
		ctx,
		pubKey,
		&rpc.GetTokenAccountsConfig{
			ProgramId: &solana.TokenProgramID,
		},
		&rpc.GetTokenAccountsOpts{
			Commitment: rpc.CommitmentFinalized,
			Encoding:   solana.EncodingJSONParsed,
		},
	)
	if err != nil {
		return nil, err
	}

	var results []*Balance
	for _, acc := range out.Value {
		parsed := acc.Account.Data.GetRawJSON().(map[string]interface{})
		info := parsed["parsed"].(map[string]interface{})["info"].(map[string]interface{})
		tokenAmount := info["tokenAmount"].(map[string]interface{})
		
		uiAmount := tokenAmount["uiAmount"].(float64)
		if uiAmount == 0 {
			continue
		}

		mint := info["mint"].(string)
		decimals := int(tokenAmount["decimals"].(float64))
		amountStr := tokenAmount["amount"].(string)
		amount, _ := new(big.Int).SetString(amountStr, 10)

		// For now, use the first 4 chars of mint as symbol if we don't have a lookup
		symbol := mint[:4]
		if mint == "Es9vMFrzaDCSTMdAhcXuzDeWvVK7UXhcrxspTS7jsX3" {
			symbol = "USDT"
		} else if mint == "EPjFWdd5AufqSSqeM2qN1xzybapC8G4wEGGkZwyTDt1v" {
			symbol = "USDC"
		}

		results = append(results, &Balance{
			Chain:    vault.ChainSolana,
			Address:  mint,
			Symbol:   symbol,
			Amount:   amount,
			Decimals: decimals,
		})
	}

	return results, nil
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
