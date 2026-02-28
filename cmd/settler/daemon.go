package main

import (
	"log"
	"os"
	"time"

	"github.com/nathfavour/settlerwallet/internal/vault"
	"github.com/spf13/cobra"
	"gopkg.in/telebot.v3"
)

func init() {
	rootCmd.AddCommand(daemonCmd)
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Starts the settlerWallet daemon (Telegram Bot).",
	Run: func(cmd *cobra.Command, args []string) {
		token := os.Getenv("TELEGRAM_BOT_TOKEN")
		if token == "" {
			log.Println("❌ Error: TELEGRAM_BOT_TOKEN is not set.")
			log.Println("Please set it using: export TELEGRAM_BOT_TOKEN=\"your_bot_token\"")
			log.Println("You can get a token from @BotFather on Telegram.")
			os.Exit(1)
		}

		pref := telebot.Settings{
			Token:  token,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		}

		b, err := telebot.NewBot(pref)
		if err != nil {
			log.Fatalf("❌ Failed to start bot: %v", err)
		}

		// 1. Initial /start command
		b.Handle("/start", func(c telebot.Context) error {
			return c.Send("Welcome to settlerWallet. Your agentic financial partner. Send /setup to begin.")
		})

		// 2. Setup (Mnemonic generation)
		b.Handle("/setup", func(c telebot.Context) error {
			mnemonic, err := vault.GenerateMnemonic()
			if err != nil {
				return c.Send("Error generating mnemonic: " + err.Error())
			}
			// In production, we'd encrypt this immediately and ask for a password.
			return c.Send("Your mnemonic (SAVE THIS!): " + mnemonic)
		})

		log.Println("settlerWallet daemon starting...")
		b.Start()
	},
}
