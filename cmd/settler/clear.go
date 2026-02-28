package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(clearCmd)
}

var clearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Factory reset: Clears all local accounts, wallets, and configuration.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("⚠️  WARNING: This will PERMANENTLY delete all local accounts, wallets, and settings.")
		fmt.Println("   Make sure you have your recovery phrases backed up if you want to restore them later.")
		fmt.Print("\nTo confirm, please type 'RESET': ")

		var confirmation string
		fmt.Scanln(&confirmation)

		if strings.ToUpper(confirmation) != "RESET" {
			fmt.Println("❌ Reset cancelled.")
			return
		}

		appDir := getAppDir()
		
		// Attempt to delete the database and config
		dbPath := getDBPath()
		configPath := getConfigPath()

		err := os.Remove(dbPath)
		if err != nil && !os.IsNotExist(err) {
			log.Printf("⚠️  Could not delete database: %v", err)
		} else {
			fmt.Println("✅ Database cleared.")
		}

		err = os.Remove(configPath)
		if err != nil && !os.IsNotExist(err) {
			log.Printf("⚠️  Could not delete config: %v", err)
		} else {
			fmt.Println("✅ Configuration cleared.")
		}

		// Optional: Try to remove the directory if empty
		os.Remove(appDir)

		fmt.Println("\n✨ Factory reset complete. Your settlerWallet is now clean.")
	},
}
