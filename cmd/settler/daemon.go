package main

import (
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/nathfavour/settlerwallet/internal/guardrail"
	"github.com/nathfavour/settlerwallet/internal/persistence"
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
			log.Fatal("❌ Error: TELEGRAM_BOT_TOKEN is not set.")
		}

		serverSecret := os.Getenv("SERVER_SECRET")
		if serverSecret == "" {
			log.Fatal("❌ Error: SERVER_SECRET is not set (used for vault encryption).")
		}

		finalDBPath := dbPath
		if finalDBPath == "" {
			configDir, err := os.UserConfigDir()
			if err != nil {
				log.Fatalf("❌ Failed to get config directory: %v", err)
			}
			appDir := fmt.Sprintf("%s/settlerwallet", configDir)
			if err := os.MkdirAll(appDir, 0700); err != nil {
				log.Fatalf("❌ Failed to create config directory: %v", err)
			}
			finalDBPath = fmt.Sprintf("%s/settler.db", appDir)
		}

		db, err := persistence.NewDB(finalDBPath)
		if err != nil {
			log.Fatalf("❌ Failed to initialize database at %s: %v", finalDBPath, err)
		}
		defer db.Close()
		log.Printf("📂 Database initialized at: %s", finalDBPath)

		engine := guardrail.NewEngine(db)

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
			return c.Send("Welcome to settlerWallet. Your multi-chain agentic financial partner.\n\n" +
				"Commands:\n" +
				"/setup - Create a new wallet\n" +
				"/address - Show your addresses\n" +
				"/limit <amount> - Set daily spend limit (in native units)\n" +
				"/balance - Check balances (WIP)")
		})

		// 2. Setup (Mnemonic generation and storage)
		b.Handle("/setup", func(c telebot.Context) error {
			uid := c.Sender().ID
			existing, err := db.GetVault(uid)
			if err == nil && existing != nil {
				return c.Send("⚠️ You already have a wallet set up. Use /address to see it.")
			}

			mnemonic, err := vault.GenerateMnemonic()
			if err != nil {
				return c.Send("Error generating mnemonic: " + err.Error())
			}

			v, err := vault.NewVault(mnemonic, strconv.FormatInt(uid, 10), serverSecret)
			if err != nil {
				return c.Send("Error creating vault: " + err.Error())
			}

			err = db.SaveVault(persistence.UserVault{
				TelegramID:    uid,
				EncryptedSeed: v.EncryptedSeed,
				Salt:          v.Salt,
			})
			if err != nil {
				return c.Send("Error saving vault: " + err.Error())
			}

			return c.Send("✅ Wallet created!\n\nYour mnemonic (SAVE THIS!): \n\n`"+mnemonic+"`", telebot.ModeMarkdown)
		})

		// 3. Address command
		b.Handle("/address", func(c telebot.Context) error {
			uid := c.Sender().ID
			uv, err := db.GetVault(uid)
			if err != nil || uv == nil {
				return c.Send("❌ No wallet found. Send /setup to create one.")
			}

			v := &vault.Vault{
				EncryptedSeed: uv.EncryptedSeed,
				Salt:          uv.Salt,
			}

			uidStr := strconv.FormatInt(uid, 10)
			bnbAcc, err := v.DeriveAccount(uidStr, serverSecret, vault.ChainBNB, 0)
			if err != nil {
				return c.Send("Error deriving BNB address: " + err.Error())
			}

			solAcc, err := v.DeriveAccount(uidStr, serverSecret, vault.ChainSolana, 0)
			if err != nil {
				return c.Send("Error deriving Solana address: " + err.Error())
			}

			return c.Send(fmt.Sprintf("📍 Your Addresses:\n\nBNB: `%s`\nSolana: `%s`", bnbAcc.Address, solAcc.Address), telebot.ModeMarkdown)
		})

		// 4. Limit command
		b.Handle("/limit", func(c telebot.Context) error {
			args := c.Args()
			if len(args) == 0 {
				return c.Send("Usage: /limit <amount_in_native_units>")
			}

			amountFloat, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				return c.Send("Invalid amount. Please provide a number.")
			}

			// Convert to 1e18 (Simplified for BNB/ETH native)
			limit := new(big.Int)
			new(big.Float).Mul(big.NewFloat(amountFloat), big.NewFloat(1e18)).Int(limit)

			if err := engine.SetLimit(c.Sender().ID, limit); err != nil {
				return c.Send("Error setting limit: " + err.Error())
			}

			return c.Send(fmt.Sprintf("✅ Daily spend limit set to %s units.", args[0]))
		})

		log.Println("settlerWallet daemon starting...")
		b.Start()
	},
}
