package blockchain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BNBClient struct {
	client *ethclient.Client
}

func NewBNBClient(rpcURL string) (*BNBClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to BNB RPC: %w", err)
	}
	return &BNBClient{client: client}, nil
}

func (c *BNBClient) GetBalance(ctx context.Context, address string) (*Balance, error) {
	account := common.HexToAddress(address)
	balance, err := c.client.BalanceAt(ctx, account, nil)
	if err != nil {
		return nil, err
	}

	return &Balance{
		Chain:    BNB,
		Address:  address,
		Symbol:   "BNB",
		Amount:   balance,
		Decimals: 18,
	}, nil
}

func (c *BNBClient) Transfer(ctx context.Context, from *DerivedKey, req Transfer) (*TransactionResult, error) {
	if from.Chain != BNB {
		return nil, fmt.Errorf("invalid chain for BNB transfer")
	}

	fromAddress := common.HexToAddress(from.Address)
	nonce, err := c.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, err
	}

	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	toAddress := common.HexToAddress(req.To)
	gasLimit := uint64(21000) // Standard for BNB transfer

	tx := types.NewTransaction(nonce, toAddress, req.Amount, gasLimit, gasPrice, nil)

	chainID, err := c.client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	privKey, err := crypto.ToECDSA(from.PrivateKey)
	if err != nil {
		return nil, err
	}

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privKey)
	if err != nil {
		return nil, err
	}

	err = c.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return nil, err
	}

	return &TransactionResult{
		Hash:    signedTx.Hash().Hex(),
		Success: true,
	}, nil
}
