package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
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

		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Failed to initialize database: %v", err)
		}
		defer db.Close()

		engine := guardrail.NewEngine(db)

		pref := telebot.Settings{
			Token:  token,
			Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
		}

		b, err := telebot.NewBot(pref)
		if err != nil {
			log.Fatalf("❌ Failed to start bot: %v", err)
		}

		// --- State Management ---
		type setupState struct {
			step     int
			password string
		}
		userStates := make(map[int64]*setupState)

		// --- Keyboards ---
		menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnWallet := menu.Text("💳 Wallet")
		btnGuardrails := menu.Text("🛡️ Guardrails")
		btnBack := menu.Text("⬅️ Back")

		menu.Reply(menu.Row(btnWallet, btnGuardrails))

		walletMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnBalance := walletMenu.Text("💰 Balances")
		btnAddress := walletMenu.Text("📍 Addresses")

		walletMenu.Reply(
			walletMenu.Row(btnBalance, btnAddress),
			walletMenu.Row(btnBack),
		)

		// --- Handlers ---
		b.Handle("/start", func(c telebot.Context) error {
			return c.Send("Welcome to settlerWallet. Secure multi-chain partner.\n\nSend /setup to begin.", menu)
		})

		b.Handle("/setup", func(c telebot.Context) error {
			uid := c.Sender().ID
			accountID := fmt.Sprintf("tg:%d", uid)
			acc, _ := db.GetAccount(accountID)
			if acc != nil {
				return c.Send("⚠️ Already set up.")
			}
			userStates[uid] = &setupState{step: 1}
			return c.Send("🔐 Enter a password to encrypt your vault:")
		})

		b.Handle(telebot.OnText, func(c telebot.Context) error {
			uid := c.Sender().ID
			state, ok := userStates[uid]
			if !ok {
				return nil
			}
			if state.step == 1 {
				state.password = c.Text()
				state.step = 2
				c.Delete()
				return c.Send("✅ Password set. Send /confirm to generate your vault.")
			}
			return nil
		})

		b.Handle("/confirm", func(c telebot.Context) error {
			uid := c.Sender().ID
			state, ok := userStates[uid]
			if !ok || state.step != 2 {
				return c.Send("❌ No pending setup.")
			}

			accountID := fmt.Sprintf("tg:%d", uid)
			salt := make([]byte, 16)
			io.ReadFull(rand.Reader, salt)

			acc := persistence.Account{
				ID:         accountID,
				Type:       persistence.AccountTelegram,
				Salt:       salt,
				Iterations: vault.KDFIterations,
			}
			db.SaveAccount(acc)

			mnemonic, _ := vault.GenerateMnemonic()
			seed := vault.GetSeedFromMnemonic(mnemonic)
			encryptedSeed, _ := vault.Encrypt(seed, state.password, salt, acc.Iterations)
			
			_, bnbAddr, _ := vault.DerivePrivateKey(seed, vault.ChainBNB, 0)
			_, solAddr, _ := vault.DerivePrivateKey(seed, vault.ChainSolana, 0)

			db.SaveWallet(persistence.Wallet{
				AccountID:     accountID,
				Name:          "BNB Main",
				Chain:         string(vault.ChainBNB),
				Address:       bnbAddr,
				EncryptedSeed: encryptedSeed,
				Salt:          make([]byte, 12),
			})

			db.SaveWallet(persistence.Wallet{
				AccountID:     accountID,
				Name:          "Solana Main",
				Chain:         string(vault.ChainSolana),
				Address:       solAddr,
				EncryptedSeed: encryptedSeed,
				Salt:          make([]byte, 12),
			})

			delete(userStates, uid)
			return c.Send(fmt.Sprintf("✅ Vault Created!\n\nMnemonic: `%s`", mnemonic), telebot.ModeMarkdown)
		})

		b.Handle(&btnWallet, func(c telebot.Context) error {
			return c.Send("Wallet Menu:", walletMenu)
		})

		b.Handle(&btnAddress, func(c telebot.Context) error {
			uid := c.Sender().ID
			wallets, _ := db.GetWallets(fmt.Sprintf("tg:%d", uid))
			if len(wallets) == 0 {
				return c.Send("❌ No wallets.")
			}
			msg := "📍 Addresses:\n"
			for _, w := range wallets {
				msg += fmt.Sprintf("- %s: `%s`\n", w.Name, w.Address)
			}
			return c.Send(msg, telebot.ModeMarkdown)
		})

		b.Handle(&btnBalance, func(c telebot.Context) error {
			uid := c.Sender().ID
			wallets, _ := db.GetWallets(fmt.Sprintf("tg:%d", uid))
			if len(wallets) == 0 {
				return c.Send("❌ No wallets.")
			}

			bnbClient, _ := blockchain.NewBNBClient("https://bsc-dataseed.binance.org")
			solClient, _ := blockchain.NewSolanaClient("https://api.mainnet-beta.solana.com")
			
			msg := "💰 Balances:\n"
			for _, w := range wallets {
				var balStr string
				if w.Chain == string(vault.ChainBNB) {
					b, _ := bnbClient.GetBalance(context.Background(), w.Address)
					balStr = utils.FormatBalance(b.Amount, 18) + " BNB"
				} else {
					b, _ := solClient.GetBalance(context.Background(), w.Address)
					balStr = utils.FormatBalance(b.Amount, 9) + " SOL"
				}
				msg += fmt.Sprintf("- %s: `%s`\n", w.Name, balStr)
			}
			return c.Send(msg, telebot.ModeMarkdown)
		})

		b.Handle(&btnBack, func(c telebot.Context) error {
			return c.Send("Main Menu:", menu)
		})

		log.Printf("settlerWallet daemon starting (Engine: %p)...", engine)
		b.Start()
	},
}
