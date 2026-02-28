package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(switchCmd)
}

var switchCmd = &cobra.Command{
	Use:   "switch",
	Short: "Switch between or list local accounts.",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		accounts, err := db.GetAccountsByType("local")
		if err != nil {
			log.Fatalf("❌ Error fetching accounts: %v", err)
		}

		if len(accounts) == 0 {
			fmt.Println("ℹ️ No local accounts found. Run 'setup' to create one.")
			return
		}

		fmt.Println("📍 Available Local Accounts:")
		for _, acc := range accounts {
			name := acc.ID[6:] // Strip "local:"
			fmt.Printf("- %s\n", name)
			
			wallets, _ := db.GetWallets(acc.ID)
			for _, w := range wallets {
				fmt.Printf("  └ %s: %s\n", w.Name, w.Address)
			}
		}

		fmt.Println("\nTo use a specific account, use the --name flag with other commands.")
		fmt.Println("Example: ./settlerwallet setup --name my-other-account")
	},
}
