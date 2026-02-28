package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"io"

	"github.com/nathfavour/settlerwallet/pkg/utils"
	"github.com/tyler-smith/go-bip39"
	"golang.org/x/crypto/pbkdf2"
)

const (
	KDFIterations = 100000
	KeyLength     = 32
)

var (
	ErrDecryptionFailed = errors.New("decryption failed: invalid password or corrupted data")
)

// DeriveKey generates a deterministic AES-256 key from a password and salt.
func DeriveKey(password string, salt []byte, iterations int) []byte {
	return pbkdf2.Key([]byte(password), salt, iterations, KeyLength, sha256.New)
}

// Encrypt encrypts data using a password and salt.
func Encrypt(data []byte, password string, salt []byte, iterations int) ([]byte, error) {
	key := DeriveKey(password, salt, iterations)
	defer utils.ZeroMemory(key)

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

	return gcm.Seal(nonce, nonce, data, nil), nil
}

// Decrypt decrypts data using a password and salt.
func Decrypt(ciphertext []byte, password string, salt []byte, iterations int) ([]byte, error) {
	key := DeriveKey(password, salt, iterations)
	defer utils.ZeroMemory(key)

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

	nonce, encryptedData := ciphertext[:nonceSize], ciphertext[nonceSize:]
	data, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, ErrDecryptionFailed
	}

	return data, nil
}

// GenerateMnemonic creates a new BIP39 mnemonic (24 words).
func GenerateMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	return bip39.NewMnemonic(entropy)
}
