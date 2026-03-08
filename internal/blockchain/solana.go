package blockchain

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"time"

	"github.com/gagliardetto/solana-go"
	ata "github.com/gagliardetto/solana-go/programs/associated-token-account"
	"github.com/gagliardetto/solana-go/programs/compute-budget"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/programs/token"
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
		var data struct {
			Parsed struct {
				Info struct {
					Mint        string `json:"mint"`
					TokenAmount struct {
						Amount   string  `json:"amount"`
						Decimals int     `json:"decimals"`
						UiAmount float64 `json:"uiAmount"`
					} `json:"tokenAmount"`
				} `json:"info"`
			} `json:"parsed"`
		}

		raw, err := acc.Account.Data.MarshalJSON()
		if err != nil {
			continue
		}

		err = json.Unmarshal(raw, &data)
		if err != nil {
			continue
		}

		if data.Parsed.Info.TokenAmount.UiAmount == 0 {
			continue
		}

		mint := data.Parsed.Info.Mint
		decimals := data.Parsed.Info.TokenAmount.Decimals
		amount, _ := new(big.Int).SetString(data.Parsed.Info.TokenAmount.Amount, 10)

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

	var instructions []solana.Instruction

	// 1. Add Priority Fees (Robustness: Essential for Mainnet)
	// We'll use a conservative default, but in a real app this might be dynamic.
	instructions = append(instructions,
		computeBudget.NewSetComputeUnitPriceInstruction(1000).Build(), // 1000 micro-lamports
		computeBudget.NewSetComputeUnitLimitInstruction(200000).Build(),
	)

	if req.Token == "" {
		// Native SOL Transfer
		instructions = append(instructions,
			system.NewTransferInstruction(
				req.Amount.Uint64(),
				fromPubKey,
				toPubKey,
			).Build(),
		)
	} else {
		// SPL Token Transfer
		mintPubKey, err := solana.PublicKeyFromBase58(req.Token)
		if err != nil {
			return nil, fmt.Errorf("invalid token mint: %w", err)
		}

		// Get source ATA
		sourceATA, _, err := solana.FindAssociatedTokenAddress(fromPubKey, mintPubKey)
		if err != nil {
			return nil, err
		}

		// Get destination ATA
		destATA, _, err := solana.FindAssociatedTokenAddress(toPubKey, mintPubKey)
		if err != nil {
			return nil, err
		}

		// Check if destination ATA exists, if not, create it
		_, err = c.client.GetAccountInfo(ctx, destATA)
		if err != nil {
			// Assume it doesn't exist (or other error, but we'll try to create it)
			instructions = append(instructions,
				ata.NewCreateInstruction(
					fromPubKey,
					toPubKey,
					mintPubKey,
				).Build(),
			)
		}

		instructions = append(instructions,
			token.NewTransferInstruction(
				req.Amount.Uint64(),
				sourceATA,
				destATA,
				fromPubKey,
				[]solana.PublicKey{},
			).Build(),
		)
	}

	recent, err := c.client.GetRecentBlockhash(ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}

	tx, err := solana.NewTransaction(
		instructions,
		recent.Value.Blockhash,
		solana.TransactionPayer(fromPubKey),
	)
	if err != nil {
		return nil, err
	}

	// 2. Simulation (Robustness: Catch errors before broadcast)
	sim, err := c.client.SimulateTransaction(ctx, tx)
	if err != nil {
		return nil, fmt.Errorf("simulation failed: %w", err)
	}
	if sim.Value.Err != nil {
		return nil, fmt.Errorf("simulation error: %v", sim.Value.Err)
	}

	// Sign
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

	// 3. Send and Confirm (Robustness: Ensure inclusion)
	sig, err := c.client.SendTransactionWithOpts(ctx, tx, rpc.TransactionOpts{
		SkipPreflight:       false,
		PreflightCommitment: rpc.CommitmentFinalized,
	})
	if err != nil {
		return nil, err
	}

	// Wait for confirmation (polling for simplicity, could use WebSocket)
	// In a real robust app, we'd have a better timeout/retry loop.
	success := false
	for i := 0; i < 30; i++ {
		status, err := c.client.GetSignatureStatuses(ctx, false, sig)
		if err == nil && len(status.Value) > 0 && status.Value[0] != nil {
			if status.Value[0].ConfirmationStatus == rpc.ConfirmationStatusConfirmed ||
				status.Value[0].ConfirmationStatus == rpc.ConfirmationStatusFinalized {
				success = true
				break
			}
			if status.Value[0].Err != nil {
				return &TransactionResult{
					Hash:    sig.String(),
					Success: false,
					Error:   fmt.Errorf("transaction failed: %v", status.Value[0].Err),
				}, nil
			}
		}
		time.Sleep(1 * time.Second)
	}

	return &TransactionResult{
		Hash:    sig.String(),
		Success: success,
	}, nil
}
