package blockchain

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/nathfavour/settlerwallet/internal/vault"
)

type PinionClient struct {
	baseURL string
	signer  *vault.DerivedKey
	client  *http.Client
}

func NewPinionClient(signer *vault.DerivedKey) *PinionClient {
	return &PinionClient{
		baseURL: "https://api.pinionos.com", // Default base URL
		signer:  signer,
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

type PinionTradeRequest struct {
	TokenIn  string `json:"tokenIn"`
	TokenOut string `json:"tokenOut"`
	Amount   string `json:"amount"`
}

type PinionTradeResponse struct {
	Tx struct {
		To   string `json:"to"`
		Data string `json:"data"`
		Value string `json:"value"`
	} `json:"tx"`
}

func (c *PinionClient) GetTradeQuote(ctx context.Context, req PinionTradeRequest) (*PinionTradeResponse, error) {
	url := fmt.Sprintf("%s/v1/trade?tokenIn=%s&tokenOut=%s&amount=%s", c.baseURL, req.TokenIn, req.TokenOut, req.Amount)
	
	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("pinion api error: %s", resp.Status)
	}

	var result PinionTradeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *PinionClient) doRequest(ctx context.Context, method, url string, body []byte) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	// Handle x402 Micropayment flow
	if resp.StatusCode == 402 {
		paymentTo := resp.Header.Get("X-402-Payment-To")
		paymentAmountStr := resp.Header.Get("X-402-Payment-Amount")
		
		if paymentTo == "" || paymentAmountStr == "" {
			return resp, nil // Let the caller handle the 402 if headers are missing
		}

		amount, _ := new(big.Int).SetString(paymentAmountStr, 10)
		
		// Sign EIP-3009 Authorization
		auth, err := c.signPayment(common.HexToAddress(paymentTo), amount)
		if err != nil {
			return nil, fmt.Errorf("failed to sign x402 payment: %w", err)
		}

		// Re-send request with authorization header
		req, _ = http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
		
		authJSON, _ := json.Marshal(auth)
		req.Header.Set("X-402-Authorization", hex.EncodeToString(authJSON))
		
		return c.client.Do(req)
	}

	return resp, nil
}

func (c *PinionClient) signPayment(to common.Address, amount *big.Int) (*EIP3009Authorization, error) {
	privKey, err := crypto.ToECDSA(c.signer.PrivateKey)
	if err != nil {
		return nil, err
	}

	from := common.HexToAddress(c.signer.Address)
	
	var nonce [32]byte
	_, _ = io.ReadFull(rand.Reader, nonce[:])

	validAfter := uint64(0)
	validBefore := uint64(time.Now().Add(1 * time.Hour).Unix())

	return SignTransferWithAuthorization(
		privKey,
		from,
		to,
		amount,
		validAfter,
		validBefore,
		nonce,
	)
}
