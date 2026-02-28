package main

import (
	"context"
	"fmt"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/nathfavour/settlerwallet/internal/blockchain"
	"github.com/nathfavour/settlerwallet/internal/guardrail"
	"github.com/nathfavour/settlerwallet/internal/persistence"
	"github.com/nathfavour/settlerwallet/internal/vault"
	"github.com/nathfavour/settlerwallet/pkg/utils"
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

		// --- Keyboards ---
		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnWallet := menu.Text("💳 Wallet")
		btnGuardrails := menu.Text("🛡️ Guardrails")
		btnStrategies := menu.Text("🤖 Strategies")
		btnSettings := menu.Text("⚙️ Settings")

		menu.Reply(
			menu.Row(btnWallet, btnGuardrails),
			menu.Row(btnStrategies, btnSettings),
		)

		walletMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnBalance := walletMenu.Text("💰 Balances")
		btnAddress := walletMenu.Text("📍 Addresses")
		btnBack := walletMenu.Text("⬅️ Back")

		walletMenu.Reply(
			walletMenu.Row(btnBalance, btnAddress),
			walletMenu.Row(btnBack),
		)

		guardMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnViewLimit := guardMenu.Text("📉 View Limit")
		btnSetLimit := guardMenu.Text("✏️ Set Limit")

		guardMenu.Reply(
			guardMenu.Row(btnViewLimit, btnSetLimit),
			guardMenu.Row(btnBack),
		)

		// --- Handlers ---

		// 1. Initial /start command
		b.Handle("/start", func(c telebot.Context) error {
			return c.Send("Welcome to settlerWallet. Your multi-chain agentic financial partner.\n\nUse the menu below to navigate.", menu)
		})

		// Menu Navigation
		b.Handle(&btnWallet, func(c telebot.Context) error {
			return c.Send("Wallet Menu:", walletMenu)
		})

		b.Handle(&btnGuardrails, func(c telebot.Context) error {
			return c.Send("Guardrails Menu:", guardMenu)
		})

		b.Handle(&btnBack, func(c telebot.Context) error {
			return c.Send("Main Menu:", menu)
		})

		// 2. Setup (Mnemonic generation and storage)
		b.Handle("/setup", func(c telebot.Context) error {
			uid := c.Sender().ID
			existing, err := db.GetVault(uid)
			if err == nil && existing != nil {
				return c.Send("⚠️ You already have a wallet set up. Use 📍 Addresses to see it.")
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

		// 3. Address command/button
		b.Handle(&btnAddress, func(c telebot.Context) error {
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

		// 4. Balance command/button
		b.Handle(&btnBalance, func(c telebot.Context) error {
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
			bnbAcc, _ := v.DeriveAccount(uidStr, serverSecret, vault.ChainBNB, 0)
			solAcc, _ := v.DeriveAccount(uidStr, serverSecret, vault.ChainSolana, 0)

			// Load clients
			bnbClient, _ := blockchain.NewBNBClient("https://bsc-dataseed.binance.org")
			solClient, _ := blockchain.NewSolanaClient("https://api.mainnet-beta.solana.com")

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			msg := "💰 Your Balances:\n\n"

			bnbBal, err := bnbClient.GetBalance(ctx, bnbAcc.Address)
			if err != nil {
				msg += "BNB: Error fetching\n"
			} else {
				msg += fmt.Sprintf("BNB: `%s BNB`\n", formatBalance(bnbBal.Amount, 18))
			}

			solBal, err := solClient.GetBalance(ctx, solAcc.Address)
			if err != nil {
				msg += "SOL: Error fetching\n"
			} else {
				msg += fmt.Sprintf("SOL: `%s SOL`\n", formatBalance(solBal.Amount, 9))
			}

			return c.Send(msg, telebot.ModeMarkdown)
		})

		// 5. Guardrail Commands
		b.Handle(&btnViewLimit, func(c telebot.Context) error {
			uid := c.Sender().ID
			rules, err := db.GetRules(uid)
			if err != nil {
				return c.Send("Error fetching rules: " + err.Error())
			}
			if rules == nil {
				return c.Send("No limit set. Defaulting to 1 unit per day.")
			}

			limit, _ := new(big.Int).SetString(rules.DailyLimit, 10)
			current, _ := new(big.Int).SetString(rules.CurrentSpend, 10)

			return c.Send(fmt.Sprintf("🛡️ Guardrail Status:\n\nDaily Limit: `%s` units\nSpent Today: `%s` units",
				formatBalance(limit, 18), formatBalance(current, 18)), telebot.ModeMarkdown)
		})

		b.Handle(&btnSetLimit, func(c telebot.Context) error {
			return c.Send("To set your daily limit, use the command:\n`/limit <amount>`\n\nExample: `/limit 0.5`", telebot.ModeMarkdown)
		})

		b.Handle("/limit", func(c telebot.Context) error {
			args := c.Args()
			if len(args) == 0 {
				return c.Send("Usage: /limit <amount_in_native_units>")
			}

			amountFloat, err := strconv.ParseFloat(args[0], 64)
			if err != nil {
				return c.Send("Invalid amount. Please provide a number.")
			}

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

func formatBalance(amount *big.Int, decimals int) string {
	f := new(big.Float).SetInt(amount)
	f.Quo(f, big.NewFloat(10).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)))
	return f.Text('f', 4)
}
