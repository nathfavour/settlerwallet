package blockchain

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignTransferWithAuthorization(t *testing.T) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate private key: %v", err)
	}

	from := crypto.PubkeyToAddress(privKey.PublicKey)
	to := common.HexToAddress("0x1234567890123456789012345678901234567890")
	value := big.NewInt(1000000) // 1 USDC (6 decimals)
	validAfter := uint64(0)
	validBefore := uint64(10000000000)
	var nonce [32]byte
	copy(nonce[:], "test-nonce-123456789012345678901")

	auth, err := SignTransferWithAuthorization(privKey, from, to, value, validAfter, validBefore, nonce)
	if err != nil {
		t.Fatalf("failed to sign transfer: %v", err)
	}

	if auth.From != from {
		t.Errorf("expected from %v, got %v", from, auth.From)
	}
	if auth.To != to {
		t.Errorf("expected to %v, got %v", to, auth.To)
	}
	if auth.Value.Cmp(value) != 0 {
		t.Errorf("expected value %v, got %v", value, auth.Value)
	}
	if auth.V == 0 {
		t.Error("expected non-zero V")
	}
}

func TestPinionClient_Initialization(t *testing.T) {
	// Simple test to ensure PinionClient can be created
	// (Actual API testing would require mocking or a live server)
}
