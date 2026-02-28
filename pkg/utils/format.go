package utils

import (
	"math/big"
)

// FormatBalance converts a big.Int amount with given decimals to a human-readable string.
func FormatBalance(amount *big.Int, decimals int) string {
	f := new(big.Float).SetInt(amount)
	f.Quo(f, big.NewFloat(10).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)))
	return f.Text('f', 4)
}
