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
	Use:   "switch [account_name]",
	Short: "Switch between or list local accounts.",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		if len(args) > 0 {
			targetName := args[0]
			accountID := fmt.Sprintf("local:%s", targetName)
			acc, _ := db.GetAccount(accountID)
			if acc == nil {
				log.Fatalf("❌ Error: Account '%s' not found.", targetName)
			}

			cfg := loadConfig()
			cfg.ActiveAccount = targetName
			if err := saveConfig(cfg); err != nil {
				log.Fatalf("❌ Error saving config: %v", err)
			}
			fmt.Printf("✅ Switched to account: %s\n", targetName)
			return
		}

		// List accounts if no arg
		accounts, err := db.GetAccountsByType("local")
		if err != nil {
			log.Fatalf("❌ Error fetching accounts: %v", err)
		}

		if len(accounts) == 0 {
			fmt.Println("ℹ️ No local accounts found. Run 'setup' to create one.")
			return
		}

		cfg := loadConfig()
		fmt.Println("📍 Available Local Accounts:")
		for _, acc := range accounts {
			name := acc.ID[6:] // Strip "local:"
			indicator := "  "
			if name == cfg.ActiveAccount {
				indicator = "👉"
			}
			fmt.Printf("%s %s\n", indicator, name)

			wallets, _ := db.GetWallets(acc.ID)
			for _, w := range wallets {
				fmt.Printf("   └ %s: %s\n", w.Name, w.Address)
			}
		}

		fmt.Println("\nTo switch account: settler switch <name>")
	},
}
