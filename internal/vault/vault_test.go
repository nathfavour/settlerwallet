package vault

import (
	"bytes"
	"testing"
)

func TestVault_Lifecycle(t *testing.T) {
	telegramID := "12345678"
	serverSecret := "ultra-secure-server-secret"

	// 1. Generate mnemonic
	mnemonic, err := GenerateMnemonic()
	if err != nil {
		t.Fatalf("failed to generate mnemonic: %v", err)
	}
	if mnemonic == "" {
		t.Fatal("generated mnemonic is empty")
	}

	// 2. Create Vault
	v, err := NewVault(mnemonic, telegramID, serverSecret)
	if err != nil {
		t.Fatalf("failed to create vault: %v", err)
	}

	// 3. Derive Account (BNB)
	dkBNB, err := v.DeriveAccount(telegramID, serverSecret, ChainBNB, 0)
	if err != nil {
		t.Fatalf("failed to derive BNB account: %v", err)
	}
	if dkBNB.Address == "" {
		t.Fatal("derived BNB address is empty")
	}
	if dkBNB.Chain != ChainBNB {
		t.Fatalf("expected BNB chain, got %s", dkBNB.Chain)
	}

	// 4. Derive Account (Solana)
	dkSol, err := v.DeriveAccount(telegramID, serverSecret, ChainSolana, 0)
	if err != nil {
		t.Fatalf("failed to derive Solana account: %v", err)
	}
	if dkSol.Address == "" {
		t.Fatal("derived Solana address is empty")
	}
	if dkSol.Chain != ChainSolana {
		t.Fatalf("expected Solana chain, got %s", dkSol.Chain)
	}

	// 5. Test Decryption with wrong credentials
	_, err = v.DeriveAccount(telegramID, "wrong-secret", ChainBNB, 0)
	if err == nil {
		t.Fatal("expected decryption to fail with wrong secret, but it succeeded")
	}

	// 6. Test Signatures
	hash := make([]byte, 32)
	for i := range hash {
		hash[i] = byte(i)
	}

	sigBNB, err := dkBNB.SignEVM(hash)
	if err != nil {
		t.Fatalf("failed to sign EVM: %v", err)
	}
	if len(sigBNB) == 0 {
		t.Fatal("EVM signature is empty")
	}

	sigSol, err := dkSol.SignSolana(hash)
	if err != nil {
		t.Fatalf("failed to sign Solana: %v", err)
	}
	if len(sigSol) == 0 {
		t.Fatal("Solana signature is empty")
	}
}

func TestVault_Deterministic(t *testing.T) {
	telegramID := "999"
	serverSecret := "secret"
	mnemonic := "abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon abandon about"

	v, _ := NewVault(mnemonic, telegramID, serverSecret)
	
	dk1, _ := v.DeriveAccount(telegramID, serverSecret, ChainBNB, 0)
	dk2, _ := v.DeriveAccount(telegramID, serverSecret, ChainBNB, 0)

	if dk1.Address != dk2.Address {
		t.Fatal("derived addresses are not deterministic")
	}

	if !bytes.Equal(dk1.PrivateKey, dk2.PrivateKey) {
		t.Fatal("derived private keys are not deterministic")
	}
}
