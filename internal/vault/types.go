package vault

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
