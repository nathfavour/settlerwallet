package blockchain

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/nathfavour/settlerwallet/internal/vault"
)

type EVMClient struct {
	client *ethclient.Client
	chain  vault.Chain
	symbol string
}

func NewEVMClient(rpcURL string, chain vault.Chain, symbol string) (*EVMClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM RPC: %w", err)
	}
	return &EVMClient{
		client: client,
		chain:  chain,
		symbol: symbol,
	}, nil
}

func (c *EVMClient) GetBalance(ctx context.Context, address string) (*Balance, error) {
	account := common.HexToAddress(address)
	balance, err := c.client.BalanceAt(ctx, account, nil)
	if err != nil {
		return nil, err
	}

	return &Balance{
		Chain:    c.chain,
		Address:  address,
		Symbol:   c.symbol,
		Amount:   balance,
		Decimals: 18,
	}, nil
}

func (c *EVMClient) GetTokenBalances(ctx context.Context, address string) ([]*Balance, error) {
	var tokens []struct {
		Symbol   string
		Address  string
		Decimals int
	}

	if c.chain == vault.ChainBNB {
		tokens = []struct {
			Symbol   string
			Address  string
			Decimals int
		}{
			{"USDT", "0x55d398326f99059fF775485246999027B3197955", 18},
			{"BUSD", "0xe9e7CEA3DedcA5984780Bafc599bD69ADd087D56", 18},
			{"USDC", "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d", 18},
		}
	} else if c.chain == vault.ChainBase {
		tokens = []struct {
			Symbol   string
			Address  string
			Decimals int
		}{
			{"USDC", "0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913", 6},
			{"WETH", "0x4200000000000000000000000000000000000006", 18},
		}
	}

	var results []*Balance
	userAddr := common.HexToAddress(address)

	for _, t := range tokens {
		tokenAddr := common.HexToAddress(t.Address)
		data := append(common.Hex2Bytes("70a08231"), common.LeftPadBytes(userAddr.Bytes(), 32)...)

		msg := ethereum.CallMsg{
			To:   &tokenAddr,
			Data: data,
		}

		res, err := c.client.CallContract(ctx, msg, nil)
		if err != nil {
			continue
		}

		bal := new(big.Int).SetBytes(res)
		if bal.Sign() > 0 {
			results = append(results, &Balance{
				Chain:    c.chain,
				Address:  t.Address,
				Symbol:   t.Symbol,
				Amount:   bal,
				Decimals: t.Decimals,
			})
		}
	}

	return results, nil
}

func (c *EVMClient) Transfer(ctx context.Context, from *vault.DerivedKey, req Transfer) (*TransactionResult, error) {
	if from.Chain != c.chain {
		return nil, fmt.Errorf("invalid chain for transfer")
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
	gasLimit := uint64(21000)

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
