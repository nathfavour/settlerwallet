package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"

	"github.com/nathfavour/settlerwallet/pkg/utils"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
)

const (
	// PBKDF2 iterations for deriving encryption key from user ID and secret.
	kdfIterations = 100000
	keyLength     = 32 // 256 bits for AES-256
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: invalid key or corrupted data")
)

// Vault handles the secure storage and derivation of user keys.
type Vault struct {
	EncryptedSeed []byte
	Salt          []byte
}

// GenerateMnemonic creates a new BIP39 mnemonic (24 words).
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}

// DeriveKey generates a deterministic AES-256 key from a Telegram ID and a server secret.
func DeriveKey(telegramID string, serverSecret string, salt []byte) []byte {
	// Use PBKDF2 to derive a strong key from the inputs.
	return pbkdf2.Key([]byte(telegramID+serverSecret), salt, kdfIterations, keyLength, sha256.New)
}

// Encrypt encrypts the master seed using the derived key.
func Encrypt(seed []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	// Encrypt the seed.
	ciphertext := gcm.Seal(nonce, nonce, seed, nil)
	return ciphertext, nil
}

// Decrypt decrypts the master seed using the derived key.
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrDecryptionFailed
	}

	nonce, encryptedSeed := ciphertext[:nonceSize], ciphertext[nonceSize:]
	seed, err := gcm.Open(nil, nonce, encryptedSeed, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return seed, nil
}

// NewVault initializes a new vault for a user.
func NewVault(mnemonic string, telegramID string, serverSecret string) (*Vault, error) {
	// 1. Convert mnemonic to seed.
	seed := bip39.NewSeed(mnemonic, "")
	defer utils.ZeroMemory(seed)

	// 2. Generate random salt.
	salt := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, err
	}

	// 3. Derive key and encrypt.
	key := DeriveKey(telegramID, serverSecret, salt)
	defer utils.ZeroMemory(key)

	encryptedSeed, err := Encrypt(seed, key)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt vault: %w", err)
	}

	return &Vault{
		EncryptedSeed: encryptedSeed,
		Salt:          salt,
	}, nil
}
