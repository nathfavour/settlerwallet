package main

import (
	"fmt"
	"log"

	"github.com/nathfavour/settlerwallet/internal/vault"
	"github.com/spf13/cobra"
)

var (
	mnemonicInput string
)

func init() {
	addressCmd.Flags().StringVarP(&mnemonicInput, "mnemonic", "m", "", "BIP39 mnemonic to derive addresses from")
	rootCmd.AddCommand(addressCmd)
}

var addressCmd = &cobra.Command{
	Use:   "address",
	Short: "Show wallet addresses derived from a mnemonic.",
	Run: func(cmd *cobra.Command, args []string) {
		if mnemonicInput == "" {
			fmt.Println("❌ Error: --mnemonic flag is required for local address derivation.")
			fmt.Println("Example: ./settlerwallet address --mnemonic \"your mnemonic here...\"")
			return
		}

		seed := vault.GetSeedFromMnemonic(mnemonicInput)

		_, bnbAddr, err := vault.DerivePrivateKey(seed, vault.ChainBNB, 0)
		if err != nil {
			log.Fatalf("❌ Error deriving BNB address: %v", err)
		}

		_, solAddr, err := vault.DerivePrivateKey(seed, vault.ChainSolana, 0)
		if err != nil {
			log.Fatalf("❌ Error deriving Solana address: %v", err)
		}

		fmt.Println("📍 Derived Addresses:")
		fmt.Printf("BNB:     %s\n", bnbAddr)
		fmt.Printf("Solana:  %s\n", solAddr)
	},
}
