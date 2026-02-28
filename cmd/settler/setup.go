package main

import (
	"fmt"
	"log"

	"github.com/nathfavour/settlerwallet/internal/vault"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Setup your settlerWallet (generate mnemonic).",
	Run: func(cmd *cobra.Command, args []string) {
		mnemonic, err := vault.GenerateMnemonic()
		if err != nil {
			log.Fatalf("❌ Error generating mnemonic: %v", err)
		}
		fmt.Println("🚀 Welcome to settlerWallet Setup!")
		fmt.Println("\nYour mnemonic is:")
		fmt.Printf("\033[1;33m%s\033[0m\n", mnemonic)
		fmt.Println("\n⚠️  SAVE THIS! Without it, you cannot recover your funds.")
		fmt.Println("⚠️  NEVER share this with anyone.")
	},
}
