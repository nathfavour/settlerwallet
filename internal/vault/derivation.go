package vault

import (
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/nathfavour/settlerwallet/pkg/utils"
	"github.com/tyler-smith/go-bip32"
)

// Chain represents a supported blockchain.
type Chain string

const (
	ChainBNB    Chain = "BNB"
	ChainSolana Chain = "Solana"
)

// DerivedKey holds the sensitive private key and metadata.
type DerivedKey struct {
	PrivateKey []byte
	Address    string
	Chain      Chain
}

// DeriveAccount derives a private key for a specific chain and index.
func (v *Vault) DeriveAccount(telegramID string, serverSecret string, chain Chain, index uint32) (*DerivedKey, error) {
	// 1. Derive encryption key to decrypt seed.
	key := DeriveKey(telegramID, serverSecret, v.Salt)
	defer utils.ZeroMemory(key)

	seed, err := Decrypt(v.EncryptedSeed, key)
	if err != nil {
		return nil, err
	}
	defer utils.ZeroMemory(seed)

	// 2. Derive master key from seed.
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, fmt.Errorf("failed to derive master key: %w", err)
	}

	switch chain {
	case ChainBNB:
		return v.deriveEVM(masterKey, index)
	case ChainSolana:
		return v.deriveSolana(seed, index)
	default:
		return nil, fmt.Errorf("unsupported chain: %s", chain)
	}
}

// deriveEVM handles BNB/Ethereum derivation: m/44'/60'/0'/0/n
func (v *Vault) deriveEVM(masterKey *bip32.Key, index uint32) (*DerivedKey, error) {
	path, err := accounts.ParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	if err != nil {
		return nil, err
	}
	
	currentKey := masterKey
	for _, child := range path {
		currentKey, err = currentKey.NewChildKey(child)
		if err != nil {
			return nil, err
		}
	}

	privateKeyBytes := currentKey.Key
	priv, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, err
	}

	address := crypto.PubkeyToAddress(priv.PublicKey).Hex()

	return &DerivedKey{
		PrivateKey: privateKeyBytes,
		Address:    address,
		Chain:      ChainBNB,
	}, nil
}

// deriveSolana handles Solana derivation: m/44'/501'/n'/0'
func (v *Vault) deriveSolana(seed []byte, index uint32) (*DerivedKey, error) {
	// Solana's standard derivation for Ed25519 is often done differently.
	// For now, we'll use a simplified version.
	// In production, we'd use BIP32-Ed25519 or SLIP-0010.
	// Using solana-go's basic keypair generation from seed for now.
	priv := solana.PrivateKeyFromSeed(seed[:32])
	
	return &DerivedKey{
		PrivateKey: priv[:],
		Address:    priv.PublicKey().String(),
		Chain:      ChainSolana,
	}, nil
}

// SignEVM signs an EVM transaction hash.
func (dk *DerivedKey) SignEVM(hash []byte) ([]byte, error) {
	if dk.Chain != ChainBNB {
		return nil, fmt.Errorf("invalid chain for EVM signature")
	}
	priv, err := crypto.ToECDSA(dk.PrivateKey)
	if err != nil {
		return nil, err
	}
	return crypto.Sign(hash, priv)
}

// SignSolana signs a Solana message.
func (dk *DerivedKey) SignSolana(message []byte) ([]byte, error) {
	if dk.Chain != ChainSolana {
		return nil, fmt.Errorf("invalid chain for Solana signature")
	}
	wallet, err := solana.WalletFromPrivateKeyBase58(solana.PrivateKey(dk.PrivateKey).String())
	if err != nil {
		return nil, err
	}
	sig, err := wallet.PrivateKey.Sign(message)
	if err != nil {
		return nil, err
	}
	return sig[:], nil
}
