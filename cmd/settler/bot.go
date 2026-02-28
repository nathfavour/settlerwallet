package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	botCmd.AddCommand(botStatusCmd)
	rootCmd.AddCommand(botCmd)
}

var botCmd = &cobra.Command{
	Use:   "bot",
	Short: "Admin tools for configuring the Telegram bot daemon.",
	Run: func(cmd *cobra.Command, args []string) {
		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		fmt.Println("🤖 Telegram Bot Configuration")
		token := readPassword("Enter your Telegram Bot Token: ")

		if token == "" {
			fmt.Println("❌ Token cannot be empty.")
			return
		}

		if err := db.SetConfig("telegram_token", token); err != nil {
			log.Fatalf("❌ Error saving token: %v", err)
		}

		fmt.Println("✅ Telegram bot token saved successfully.")
		fmt.Println("\n⚠️  Press Enter to clear the screen (and scrollback) and exit.")
		fmt.Scanln()
		fmt.Print("\033[H\033[2J\033[3J")
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
			fmt.Println("Please run: settler bot")
		} else {
			masked := token[:4] + "..." + token[len(token)-4:]
			fmt.Printf("🤖 Bot Status: CONFIGURED (Token: %s)\n", masked)
		}
	},
}
