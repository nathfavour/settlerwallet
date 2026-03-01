package vault

import (
	"crypto/ed25519"
	"fmt"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
)

// DerivePrivateKey derives a private key from a seed for a specific chain and index.
func DerivePrivateKey(seed []byte, chain Chain, index uint32) ([]byte, string, error) {
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return nil, "", fmt.Errorf("failed to derive master key: %w", err)
	}

	switch chain {
	case ChainBNB, ChainBase:
		return deriveEVM(masterKey, index)
	case ChainSolana:
		return deriveSolana(seed, index)
	default:
		return nil, "", fmt.Errorf("unsupported chain: %s", chain)
	}
}

func deriveEVM(masterKey *bip32.Key, index uint32) ([]byte, string, error) {
	path, err := accounts.ParseDerivationPath(fmt.Sprintf("m/44'/60'/0'/0/%d", index))
	if err != nil {
		return nil, "", err
	}

	currentKey := masterKey
	for _, child := range path {
		currentKey, err = currentKey.NewChildKey(child)
		if err != nil {
			return nil, "", err
		}
	}

	priv, err := crypto.ToECDSA(currentKey.Key)
	if err != nil {
		return nil, "", err
	}

	address := crypto.PubkeyToAddress(priv.PublicKey).Hex()
	return currentKey.Key, address, nil
}

func deriveSolana(seed []byte, index uint32) ([]byte, string, error) {
	// Simple Ed25519 derivation from seed for now
	priv := ed25519.NewKeyFromSeed(seed[:32])
	address := solana.PrivateKey(priv).PublicKey().String()
	return priv, address, nil
}

// SignTransaction signs a transaction hash based on the chain.
func SignTransaction(privateKey []byte, chain Chain, data []byte) ([]byte, error) {
	switch chain {
	case ChainBNB, ChainBase:
		priv, err := crypto.ToECDSA(privateKey)
		if err != nil {
			return nil, err
		}
		return crypto.Sign(data, priv)
	case ChainSolana:
		wallet, err := solana.WalletFromPrivateKeyBase58(solana.PrivateKey(privateKey).String())
		if err != nil {
			return nil, err
		}
		sig, err := wallet.PrivateKey.Sign(data)
		if err != nil {
			return nil, err
		}
		return sig[:], nil
	default:
		return nil, fmt.Errorf("unsupported chain for signing")
	}
}

// SignEVM signs a hash using the EVM private key.
func (dk *DerivedKey) SignEVM(hash []byte) ([]byte, error) {
	if dk.Chain != ChainBNB && dk.Chain != ChainBase {
		return nil, fmt.Errorf("invalid chain for EVM signing")
	}
	return SignTransaction(dk.PrivateKey, dk.Chain, hash)
}

// SignSolana signs a hash using the Solana private key.
func (dk *DerivedKey) SignSolana(hash []byte) ([]byte, error) {
	if dk.Chain != ChainSolana {
		return nil, fmt.Errorf("invalid chain for Solana signing")
	}
	return SignTransaction(dk.PrivateKey, dk.Chain, hash)
}

// GetSeedFromMnemonic converts a mnemonic to a seed.
func GetSeedFromMnemonic(mnemonic string) []byte {
	return bip39.NewSeed(mnemonic, "")
}
