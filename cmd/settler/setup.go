package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"syscall"

	"github.com/nathfavour/settlerwallet/internal/persistence"
	"github.com/nathfavour/settlerwallet/internal/vault"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	accountNameFlag string
)

func init() {
	setupCmd.Flags().StringVarP(&accountNameFlag, "name", "n", "", "Name of the local account to create/manage")
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup [account_name]",
	Short: "Setup a local account and wallet.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		name := "default"
		if len(args) > 0 {
			name = args[0]
		} else if accountNameFlag != "" {
			name = accountNameFlag
		}

		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		accountID := fmt.Sprintf("local:%s", name)
		acc, err := db.GetAccount(accountID)

		if acc != nil {
			fmt.Printf("ℹ️ Account '%s' already exists.\n", name)
			wallets, _ := db.GetWallets(accountID)
			if len(wallets) > 0 {
				fmt.Println("\nExisting Wallets:")
				for _, w := range wallets {
					fmt.Printf("- %s: %s (%s)\n", w.Name, w.Address, w.Chain)
				}
			}

			fmt.Print("\nDo you want to add a new wallet to this account? (y/n): ")
			var resp string
			fmt.Scanln(&resp)
			if strings.ToLower(resp) != "y" {
				return
			}
		}

		// Setup flow
		fmt.Printf("\n🚀 Setting up account: %s\n", name)
		password := readPassword("Enter account encryption password: ")
		confirm := readPassword("Confirm password: ")

		if password != confirm {
			log.Fatal("❌ Passwords do not match.")
		}

		// If new account, generate salt and save
		if acc == nil {
			salt := make([]byte, 16)
			io.ReadFull(rand.Reader, salt)
			acc = &persistence.Account{
				ID:         accountID,
				Type:       persistence.AccountLocal,
				Salt:       salt,
				Iterations: vault.KDFIterations,
			}
			if err := db.SaveAccount(*acc); err != nil {
				log.Fatalf("❌ Error saving account: %v", err)
			}
			fmt.Println("✅ Account created and encrypted.")
		}

		// Create Wallet
		mnemonic, err := vault.GenerateMnemonic()
		if err != nil {
			log.Fatalf("❌ Error generating mnemonic: %v", err)
		}

		seed := vault.GetSeedFromMnemonic(mnemonic)
		// Derive initial addresses
		_, bnbAddr, _ := vault.DerivePrivateKey(seed, vault.ChainBNB, 0)
		_, solAddr, _ := vault.DerivePrivateKey(seed, vault.ChainSolana, 0)

		// Encrypt seed and mnemonic with password
		encryptedSeed, err := vault.Encrypt(seed, password, acc.Salt, acc.Iterations)
		if err != nil {
			log.Fatalf("❌ Encryption error (seed): %v", err)
		}
		encryptedMnemonic, err := vault.Encrypt([]byte(mnemonic), password, acc.Salt, acc.Iterations)
		if err != nil {
			log.Fatalf("❌ Encryption error (mnemonic): %v", err)
		}

		walletSalt := make([]byte, 12) // Nonce for GCM
		io.ReadFull(rand.Reader, walletSalt)

		existingWallets, _ := db.GetWallets(accountID)
		walletCount := len(existingWallets) / 2 // Each setup adds 2 wallets

		// Save BNB Wallet
		db.SaveWallet(persistence.Wallet{
			AccountID:         accountID,
			Name:              fmt.Sprintf("BNB-%d", walletCount+1),
			Chain:             string(vault.ChainBNB),
			Address:           bnbAddr,
			EncryptedSeed:     encryptedSeed,
			EncryptedMnemonic: encryptedMnemonic,
			Salt:              walletSalt,
		})

		// Save Solana Wallet
		db.SaveWallet(persistence.Wallet{
			AccountID:         accountID,
			Name:              fmt.Sprintf("SOL-%d", walletCount+1),
			Chain:             string(vault.ChainSolana),
			Address:           solAddr,
			EncryptedSeed:     encryptedSeed,
			EncryptedMnemonic: encryptedMnemonic,
			Salt:              walletSalt,
		})

		fmt.Println("\n✅ Wallets successfully created and sandboxed!")
		fmt.Printf("BNB:     %s\n", bnbAddr)
		fmt.Printf("Solana:  %s\n", solAddr)
		fmt.Println("\n⚠️  SAVE THIS MNEMONIC (IT IS YOUR ONLY BACKUP):")
		fmt.Printf("\033[1;33m%s\033[0m\n", mnemonic)
	},
}

func readPassword(prompt string) string {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("❌ Error reading password: %v", err)
	}
	fmt.Println()
	return string(bytePassword)
}

func initDB() (*persistence.DB, error) {
	configDir, _ := os.UserConfigDir()
	appDir := fmt.Sprintf("%s/settlerwallet", configDir)
	os.MkdirAll(appDir, 0700)
	dbPath := fmt.Sprintf("%s/settler.db", appDir)
	return persistence.NewDB(dbPath)
}
