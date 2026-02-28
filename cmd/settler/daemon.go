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
		walletMenu.Reply(walletMenu.Row(btnBalance, btnAddress), walletMenu.Row(btnBack))

		// --- Helper: Resolve Account ---
		getAccountID := func(tgID int64) string {
			// Check user preference first
			pref, _ := db.GetUserConfig(tgID, "active_account")
			if pref != "" {
				return pref
			}
			// Default to linked local account if it exists
			acc, _ := db.GetAccountByLinkedTGID(tgID)
			if acc != nil {
				return acc.ID
			}
			// Default to native TG account
			return fmt.Sprintf("tg:%d", tgID)
		}

		// --- Handlers ---
		b.Handle("/start", func(c telebot.Context) error {
			uid := c.Sender().ID
			lr, _ := db.GetLinkRequestByTGID(uid)
			if lr != nil && lr.ExpiresAt > time.Now().Unix() {
				return c.Send(fmt.Sprintf("🔗 Link request detected for local account '%s'.\n\nYour verification code is: `%s` (Expires in 10m)\n\nRun `/link %s` to complete the link.", 
					lr.AccountID[6:], lr.Code, lr.Code), telebot.ModeMarkdown)
			}

			accountID := getAccountID(uid)
			acc, _ := db.GetAccount(accountID)
			if acc == nil {
				return c.Send("Welcome! You don't have an account yet. Send /setup to create one or link a local account using `settler link` from your CLI.", menu)
			}

			welcomeMsg := "Welcome back!"
			if acc.Type == persistence.AccountLocal {
				welcomeMsg = fmt.Sprintf("Welcome back! Linked to local account: `%s`", acc.ID[6:])
			}
			return c.Send(welcomeMsg, menu, telebot.ModeMarkdown)
		})

		b.Handle("/link", func(c telebot.Context) error {
			uid := c.Sender().ID
			args := c.Args()
			if len(args) == 0 {
				return c.Send("❌ Please provide the verification code: `/link <code>`", telebot.ModeMarkdown)
			}
			code := args[0]

			lr, _ := db.GetLinkRequestByTGID(uid)
			if lr == nil || lr.Code != code || lr.ExpiresAt < time.Now().Unix() {
				return c.Send("❌ Invalid or expired verification code.")
			}

			acc, _ := db.GetAccount(lr.AccountID)
			if acc == nil {
				return c.Send("❌ Error: Local account not found.")
			}

			acc.LinkedTGID = uid
			if err := db.SaveAccount(*acc); err != nil {
				return c.Send("❌ Error saving link.")
			}

			db.DeleteLinkRequest(lr.AccountID)
			db.SetUserConfig(uid, "active_account", lr.AccountID)

			return c.Send(fmt.Sprintf("✅ Successfully linked to local account: `%s`", lr.AccountID[6:]), telebot.ModeMarkdown)
		})

		b.Handle("/switch", func(c telebot.Context) error {
			uid := c.Sender().ID

			// Find all accounts for this user
			nativeID := fmt.Sprintf("tg:%d", uid)
			nativeAcc, _ := db.GetAccount(nativeID)

			linkedAcc, _ := db.GetAccountByLinkedTGID(uid)

			var options []string
			if nativeAcc != nil {
				options = append(options, nativeID)
			}
			if linkedAcc != nil {
				options = append(options, linkedAcc.ID)
			}

			if len(options) < 2 {
				return c.Send("ℹ️ You only have one account configured. Setup another or link a local account.")
			}

			args := c.Args()
			if len(args) == 0 {
				active := getAccountID(uid)
				msg := "🔄 **Switch Account**\n\n"
				for _, opt := range options {
					indicator := "  "
					if opt == active {
						indicator = "👉"
					}
					msg += fmt.Sprintf("%s `%s`\n", indicator, opt)
				}
				msg += "\nUsage: `/switch <account_id>`"
				return c.Send(msg, telebot.ModeMarkdown)
			}

			target := args[0]
			valid := false
			for _, opt := range options {
				if opt == target {
					valid = true
					break
				}
			}

			if !valid {
				return c.Send("❌ Invalid account ID.")
			}

			db.SetUserConfig(uid, "active_account", target)
			return c.Send(fmt.Sprintf("✅ Switched to account: `%s`", target), telebot.ModeMarkdown)
		})

		b.Handle("/setup", func(c telebot.Context) error {
			accountID := getAccountID(c.Sender().ID)
			acc, _ := db.GetAccount(accountID)
			if acc != nil {
				return c.Send("⚠️ Your account is already set up.")
			}
			userStates[c.Sender().ID] = &setupState{step: 1}
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
			encryptedMnemonic, _ := vault.Encrypt([]byte(mnemonic), state.password, salt, acc.Iterations)
			
			_, bnbAddr, _ := vault.DerivePrivateKey(seed, vault.ChainBNB, 0)
			_, solAddr, _ := vault.DerivePrivateKey(seed, vault.ChainSolana, 0)

			walletSalt := make([]byte, 12)
			io.ReadFull(rand.Reader, walletSalt)

			db.SaveWallet(persistence.Wallet{
				AccountID:         accountID,
				Name:              "BNB Main",
				Chain:             string(vault.ChainBNB),
				Address:           bnbAddr,
				EncryptedSeed:     encryptedSeed,
				EncryptedMnemonic: encryptedMnemonic,
				Salt:              walletSalt,
			})

			db.SaveWallet(persistence.Wallet{
				AccountID:         accountID,
				Name:              "Solana Main",
				Chain:             string(vault.ChainSolana),
				Address:           solAddr,
				EncryptedSeed:     encryptedSeed,
				EncryptedMnemonic: encryptedMnemonic,
				Salt:              walletSalt,
			})

			delete(userStates, uid)
			return c.Send(fmt.Sprintf("✅ Vault Created!\n\nMnemonic: `%s`", mnemonic), telebot.ModeMarkdown)
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
