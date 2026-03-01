package blockchain

import (
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
)

// EIP3009Authorization represents the data for a TransferWithAuthorization call.
type EIP3009Authorization struct {
	From          common.Address
	To            common.Address
	Value         *big.Int
	ValidAfter    uint64
	ValidBefore   uint64
	Nonce         [32]byte
	V             uint8
	R             [32]byte
	S             [32]byte
}

var (
	// USDC on Base details
	USDCAddressBase = common.HexToAddress("0x833589fCD6eDb6E08f4c7C32D4f71b54bdA02913")
	BaseChainID     = big.NewInt(8453)
)

// SignTransferWithAuthorization signs a transfer with authorization for EIP-3009 (USDC).
func SignTransferWithAuthorization(
	privKey *ecdsa.PrivateKey,
	from common.Address,
	to common.Address,
	value *big.Int,
	validAfter uint64,
	validBefore uint64,
	nonce [32]byte,
) (*EIP3009Authorization, error) {
	domainSeparator, err := getUSDCBaseDomainSeparator()
	if err != nil {
		return nil, err
	}

	// TransferWithAuthorization(address from,address to,uint256 value,uint256 validAfter,uint256 validBefore,bytes32 nonce)
	typeHash := crypto.Keccak256Hash([]byte("TransferWithAuthorization(address from,address to,uint256 value,uint256 validAfter,uint256 validBefore,bytes32 nonce)"))

	dataHash := crypto.Keccak256Hash(
		typeHash.Bytes(),
		common.LeftPadBytes(from.Bytes(), 32),
		common.LeftPadBytes(to.Bytes(), 32),
		common.LeftPadBytes(value.Bytes(), 32),
		common.LeftPadBytes(new(big.Int).SetUint64(validAfter).Bytes(), 32),
		common.LeftPadBytes(new(big.Int).SetUint64(validBefore).Bytes(), 32),
		nonce[:],
	)

	msg := fmt.Sprintf("\x19\x01%s%s", string(domainSeparator.Bytes()), string(dataHash.Bytes()))
	msgHash := crypto.Keccak256Hash([]byte(msg))

	sig, err := crypto.Sign(msgHash.Bytes(), privKey)
	if err != nil {
		return nil, err
	}

	auth := &EIP3009Authorization{
		From:        from,
		To:          to,
		Value:       value,
		ValidAfter:  validAfter,
		ValidBefore: validBefore,
		Nonce:       nonce,
		V:           sig[64] + 27,
	}
	copy(auth.R[:], sig[:32])
	copy(auth.S[:], sig[32:64])

	return auth, nil
}

func getUSDCBaseDomainSeparator() (common.Hash, error) {
	// EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)
	domainTypeHash := crypto.Keccak256Hash([]byte("EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"))
	
	nameHash := crypto.Keccak256Hash([]byte("USD Coin"))
	versionHash := crypto.Keccak256Hash([]byte("2"))
	
	return crypto.Keccak256Hash(
		domainTypeHash.Bytes(),
		nameHash.Bytes(),
		versionHash.Bytes(),
		math.U256Bytes(BaseChainID),
		common.LeftPadBytes(USDCAddressBase.Bytes(), 32),
	), nil
}
