package main

import (
	"log"
	"os"
	"time"

	"github.com/nathfavour/settlerwallet/internal/nexus"
	"github.com/nathfavour/settlerwallet/internal/vault"
	"gopkg.in/telebot.v3"
)

func main() {
	pref := telebot.Settings{
		Token:  os.Getenv("TELEGRAM_BOT_TOKEN"),
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	b, err := telebot.NewBot(pref)
	if err != nil {
		log.Fatal(err)
	}

	n := nexus.NewNexus()

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
}
