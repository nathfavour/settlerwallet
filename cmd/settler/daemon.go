package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/nathfavour/settlerwallet/internal/blockchain"
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
		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Failed to initialize database: %v", err)
		}
		defer db.Close()

		token, _ := db.GetConfig("telegram_token")
		if token == "" {
			token = os.Getenv("TELEGRAM_BOT_TOKEN")
		}

		if token == "" {
			log.Fatal("❌ Error: Bot token not set. Use 'bot set-token' or TELEGRAM_BOT_TOKEN env.")
		}

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
		btnBack := menu.Text("⬅️ Back")
		menu.Reply(menu.Row(btnWallet, btnGuardrails))

		walletMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		btnBalance := walletMenu.Text("💰 Balances")
		btnAddress := walletMenu.Text("📍 Addresses")
		walletMenu.Reply(walletMenu.Row(btnBalance, btnAddress), walletMenu.Row(btnBack))

		// --- Helper: Resolve Account ---
		getAccountID := func(tgID int64) string {
			acc, _ := db.GetAccountByLinkedTGID(tgID)
			if acc != nil {
				return acc.ID
			}
			return fmt.Sprintf("tg:%d", tgID)
		}

		// --- Handlers ---
		b.Handle("/start", func(c telebot.Context) error {
			uid := c.Sender().ID
			lr, _ := db.GetLinkRequestByTGID(uid)
			if lr != nil && lr.ExpiresAt > time.Now().Unix() {
				return c.Send(fmt.Sprintf("🔗 Link request detected for local account '%s'.\n\nYour verification code is: `%s` (Expires in 10m)", 
					lr.AccountID[6:], lr.Code), telebot.ModeMarkdown)
			}

			accountID := getAccountID(uid)
			acc, _ := db.GetAccount(accountID)
			if acc == nil {
				return c.Send("Welcome! You don't have an account yet. Send /setup to create one.", menu)
			}

			welcomeMsg := "Welcome back!"
			if acc.Type == persistence.AccountLocal {
				welcomeMsg = fmt.Sprintf("Welcome back! Linked to local account: `%s`", acc.ID[6:])
			}
			return c.Send(welcomeMsg, menu, telebot.ModeMarkdown)
		})

		b.Handle("/setup", func(c telebot.Context) error {
			accountID := getAccountID(c.Sender().ID)
			acc, _ := db.GetAccount(accountID)
			if acc != nil {
				return c.Send("⚠️ Your account is already set up.")
			}
			return c.Send("Please use the password flow to setup your native bot account.")
		})

		b.Handle(&btnWallet, func(c telebot.Context) error {
			return c.Send("Wallet Menu:", walletMenu)
		})

		b.Handle(&btnAddress, func(c telebot.Context) error {
			accountID := getAccountID(c.Sender().ID)
			wallets, _ := db.GetWallets(accountID)
			if len(wallets) == 0 {
				return c.Send("❌ No wallets found.")
			}
			msg := "📍 Addresses:\n"
			for _, w := range wallets {
				msg += fmt.Sprintf("- %s: `%s`\n", w.Name, w.Address)
			}
			return c.Send(msg, telebot.ModeMarkdown)
		})

		b.Handle(&btnBalance, func(c telebot.Context) error {
			accountID := getAccountID(c.Sender().ID)
			wallets, _ := db.GetWallets(accountID)
			if len(wallets) == 0 {
				return c.Send("❌ No wallets found.")
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

		log.Println("settlerWallet daemon starting...")
		b.Start()
	},
}
