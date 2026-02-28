package main

import (
	"fmt"
	"log"
	"os"

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
			fmt.Println("Example: ./settler address --mnemonic \"your mnemonic here...\"")
			return
		}

		serverSecret := os.Getenv("SERVER_SECRET")
		if serverSecret == "" {
			serverSecret = "local-dev-secret" // Default for local use
		}

		// Use a fixed "local" ID for CLI-based derivation
		localID := "local-user"

		v, err := vault.NewVault(mnemonicInput, localID, serverSecret)
		if err != nil {
			log.Fatalf("❌ Failed to create vault: %v", err)
		}

		bnbAcc, err := v.DeriveAccount(localID, serverSecret, vault.ChainBNB, 0)
		if err != nil {
			log.Fatalf("❌ Error deriving BNB address: %v", err)
		}

		solAcc, err := v.DeriveAccount(localID, serverSecret, vault.ChainSolana, 0)
		if err != nil {
			log.Fatalf("❌ Error deriving Solana address: %v", err)
		}

		fmt.Println("📍 Derived Addresses:")
		fmt.Printf("BNB:     %s\n", bnbAcc.Address)
		fmt.Printf("Solana:  %s\n", solAcc.Address)
	},
}
