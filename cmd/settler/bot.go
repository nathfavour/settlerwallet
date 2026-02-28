package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	botCmd.AddCommand(botSetTokenCmd)
	botCmd.AddCommand(botStatusCmd)
	rootCmd.AddCommand(botCmd)
}

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Admin tools for configuring the Telegram bot daemon.",
}

var botSetTokenCmd = &cobra.Command{
	Use:   "set-token <token>",
	Short: "Sets the Telegram bot token for the daemon.",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		token := args[0]
		if err := db.SetConfig("telegram_token", token); err != nil {
			log.Fatalf("❌ Error saving token: %v", err)
		}

		fmt.Println("✅ Telegram bot token saved successfully to database.")
		fmt.Println("You can now start the bot using: ./settlerwallet daemon")
	},
}

var botStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Shows the current bot configuration status.",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		token, _ := db.GetConfig("telegram_token")
		if token == "" {
			fmt.Println("🤖 Bot Status: NOT CONFIGURED")
			fmt.Println("Please run: ./settlerwallet bot set-token <token>")
		} else {
			masked := token[:4] + "..." + token[len(token)-4:]
			fmt.Printf("🤖 Bot Status: CONFIGURED (Token: %s)\n", masked)
		}
	},
}
