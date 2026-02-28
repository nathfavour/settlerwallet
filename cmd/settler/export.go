package main

import (
	"encoding/hex"
	"fmt"
	"log"

	"github.com/nathfavour/settlerwallet/internal/vault"
	"github.com/spf13/cobra"
)

var (
	exportPhrase  bool
	exportSeed    bool
	exportAddr    bool
	exportNetwork string
)

func init() {
	exportCmd.Flags().StringVarP(&accountNameFlag, "name", "n", "default", "Name of the local account to export")
	exportCmd.Flags().BoolVar(&exportPhrase, "phrase", false, "Export the 24-word recovery phrase")
	exportCmd.Flags().BoolVar(&exportSeed, "seed", false, "Export the 512-bit binary seed (hex)")
	exportCmd.Flags().BoolVar(&exportAddr, "address", false, "Export public addresses")
	exportCmd.Flags().StringVar(&exportNetwork, "network", "", "Filter by network (e.g., BNB, SOL)")
	rootCmd.AddCommand(exportCmd)
}

var exportCmd = &cobra.Command{
	Use:   "export",
	Short: "Securely export account data.",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		accountID := fmt.Sprintf("local:%s", accountNameFlag)
		acc, _ := db.GetAccount(accountID)
		if acc == nil {
			log.Fatalf("❌ Error: Account '%s' not found.", accountNameFlag)
		}

		wallets, err := db.GetWallets(accountID)
		if err != nil || len(wallets) == 0 {
			log.Fatal("❌ No wallets found for this account.")
		}

		fmt.Printf("🔐 Exporting data for account: %s\n", accountNameFlag)
		password := readPassword("Enter account password: ")

		fmt.Println("\n--- Export Results ---")

		for _, w := range wallets {
			if exportNetwork != "" && w.Chain != exportNetwork {
				continue
			}

			fmt.Printf("\nWallet: %s (%s)\n", w.Name, w.Chain)

			if exportAddr {
				fmt.Printf("  Address: %s\n", w.Address)
			}

			if exportPhrase || exportSeed {
				// Decrypt Mnemonic
				mnemonicBytes, err := vault.Decrypt(w.EncryptedMnemonic, password, acc.Salt, acc.Iterations)
				if err != nil {
					log.Fatalf("❌ Decryption failed for %s: %v", w.Name, err)
				}
				mnemonic := string(mnemonicBytes)

				if exportPhrase {
					fmt.Printf("  Phrase:  \033[1;33m%s\033[0m\n", mnemonic)
				}

				if exportSeed {
					seed := vault.GetSeedFromMnemonic(mnemonic)
					fmt.Printf("  Seed:    %s\n", hex.EncodeToString(seed))
				}
			}
		}

		if exportPhrase {
			fmt.Print("\n⚠️  Press Enter to clear the screen and exit.")
			fmt.Scanln()
			fmt.Print("\033[H\033[2J") // ANSI clear screen
		}
	},
}
