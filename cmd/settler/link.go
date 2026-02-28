package main

import (
	"crypto/rand"
	"fmt"
	"log"
	"time"

	"github.com/nathfavour/settlerwallet/internal/persistence"
	"github.com/spf13/cobra"
)

var (
	tgID        int64
	accountName string
)

func init() {
	linkCmd.Flags().StringVarP(&accountName, "name", "n", "default", "Name of the local account to link")
	linkCmd.Flags().Int64VarP(&tgID, "to", "t", 0, "Telegram User ID to link to")
	rootCmd.AddCommand(linkCmd)
	rootCmd.AddCommand(unlinkCmd)
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Link a local account to a Telegram UID.",
	Run: func(cmd *cobra.Command, args []string) {
		if tgID == 0 {
			log.Fatal("❌ Error: --to (Telegram ID) is required.")
		}

		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		accountID := fmt.Sprintf("local:%s", accountName)
		acc, _ := db.GetAccount(accountID)
		if acc == nil {
			log.Fatalf("❌ Error: Local account '%s' not found. Run 'setup' first.", accountName)
		}

		if acc.LinkedTGID != 0 {
			log.Fatalf("❌ Error: Account '%s' is already linked to TG ID %d.", accountName, acc.LinkedTGID)
		}

		// Generate random 6-char code
		code := generateRandomCode(6)
		err = db.CreateLinkRequest(persistence.LinkRequest{
			AccountID: accountID,
			TGID:      tgID,
			Code:      code,
			ExpiresAt: time.Now().Add(10 * time.Minute).Unix(),
		})
		if err != nil {
			log.Fatalf("❌ Error creating link request: %v", err)
		}

		fmt.Printf("🚀 Link request initiated for account '%s' to TG ID %d.\n", accountName, tgID)
		fmt.Println("\nNext Steps:")
		fmt.Println("1. Open your Telegram Bot.")
		fmt.Printf("2. Run the command: /link %s\n", code)
		fmt.Println("\nThis code will expire in 10 minutes.")
	},
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink",
	Short: "Unlink a local account from Telegram.",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		accountID := fmt.Sprintf("local:%s", accountName)
		acc, _ := db.GetAccount(accountID)
		if acc == nil {
			log.Fatalf("❌ Error: Local account '%s' not found.", accountName)
		}

		acc.LinkedTGID = 0
		if err := db.SaveAccount(*acc); err != nil {
			log.Fatalf("❌ Error unlinking account: %v", err)
		}
		fmt.Printf("✅ Success! Account '%s' has been unlinked from Telegram.\n", accountName)
	},
}

func generateRandomCode(n int) string {
	const letters = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	b := make([]byte, n)
	rand.Read(b)
	for i := range b {
		b[i] = letters[int(b[i])%len(letters)]
	}
	return string(b)
}
