package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/nathfavour/settlerwallet/internal/blockchain"
	"github.com/nathfavour/settlerwallet/internal/persistence"
	"github.com/nathfavour/settlerwallet/internal/vault"
	"github.com/nathfavour/settlerwallet/pkg/utils"
	"github.com/spf13/cobra"
	"gopkg.in/telebot.v3"
)

func init() {
	rootCmd.AddCommand(daemonStartCmd)
	rootCmd.AddCommand(daemonStopCmd)
	rootCmd.AddCommand(daemonStatusCmd)
}

// isProcessRunning checks if a process with the given PID is actually alive.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, sending signal 0 doesn't kill the process but checks for its existence.
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Starts the daemon in the background.",
	Run: func(cmd *cobra.Command, args []string) {
		pidPath := getPIDPath()
		if data, err := os.ReadFile(pidPath); err == nil {
			if pid, _ := strconv.Atoi(string(data)); isProcessRunning(pid) {
				fmt.Printf("❌ Daemon is already running (PID: %d).\n", pid)
				return
			}
			// File exists but process is dead, clean it up
			os.Remove(pidPath)
		}

		logPath := getLogPath()
		logFile, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Fatalf("❌ Failed to open log file: %v", err)
		}

		// Re-execute the binary with the -f flag
		executable, _ := os.Executable()
		child := exec.Command(executable, "-f")
		child.Stdout = logFile
		child.Stderr = logFile
		child.Stdin = nil
		child.SysProcAttr = &syscall.SysProcAttr{
			Setsid: true, // Create new session to detach from terminal
		}

		if err := child.Start(); err != nil {
			log.Fatalf("❌ Failed to start background process: %v", err)
		}

		// Save PID
		os.WriteFile(pidPath, []byte(strconv.Itoa(child.Process.Pid)), 0644)
		fmt.Printf("✅ settlerWallet daemon started in background (PID: %d).\n", child.Process.Pid)
		fmt.Printf("📂 Logs: %s\n", logPath)
		os.Exit(0) // Force parent to exit immediately
	},
}

var daemonStopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stops the background daemon.",
	Run: func(cmd *cobra.Command, args []string) {
		pidPath := getPIDPath()
		data, err := os.ReadFile(pidPath)
		if err != nil {
			fmt.Println("❌ Daemon is not running (PID file not found).")
			return
		}

		pid, _ := strconv.Atoi(string(data))
		if !isProcessRunning(pid) {
			fmt.Printf("ℹ️  Process %d is already dead. Cleaning up stale PID file.\n", pid)
			os.Remove(pidPath)
			return
		}

		process, err := os.FindProcess(pid)
		if err != nil {
			fmt.Printf("❌ Failed to find process %d: %v\n", pid, err)
			os.Remove(pidPath)
			return
		}

		fmt.Printf("Stopping daemon (PID: %d)... ", pid)
		if err := process.Signal(syscall.SIGTERM); err != nil {
			fmt.Printf("failed: %v\n", err)
			return
		}

		// Wait for PID file to be removed by the child or timeout
		for i := 0; i < 5; i++ {
			if !isProcessRunning(pid) {
				fmt.Println("stopped.")
				os.Remove(pidPath)
				return
			}
			time.Sleep(500 * time.Millisecond)
		}
		
		fmt.Println("force killing...")
		process.Kill()
		os.Remove(pidPath)
		fmt.Println("✅ Daemon stopped.")
	},
}

var daemonStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Checks the status of the background daemon.",
	Run: func(cmd *cobra.Command, args []string) {
		pidPath := getPIDPath()
		data, err := os.ReadFile(pidPath)
		if err != nil {
			fmt.Println("🤖 Daemon Status: NOT RUNNING")
			return
		}

		pid, _ := strconv.Atoi(string(data))
		if !isProcessRunning(pid) {
			fmt.Printf("🤖 Daemon Status: STALE (PID: %d exists but process is dead)\n", pid)
			fmt.Println("ℹ️  Run 'settlerwallet stop' to clean up.")
			return
		}

		fmt.Printf("🤖 Daemon Status: RUNNING (PID: %d)\n", pid)
		fmt.Printf("📂 Logs: %s\n", getLogPath())
	},
}

func runDaemon() {
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

	// --- Signal Handling for PID cleanup ---
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		log.Println("Shutting down gracefully...")
		b.Stop()
		os.Remove(getPIDPath())
		os.Exit(0)
	}()

	// --- State Management ---
	type setupState struct {
		step     int
		action   string // "setup", "send", "limit", "slippage"
		data     map[string]string
		password string
	}
	userStates := make(map[int64]*setupState)

	// --- Keyboards ---
	menu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnWallet := menu.Text("💳 Wallet")
	btnGuardrails := menu.Text("🛡️ Guardrails")
	btnAgents := menu.Text("🤖 Agents")
	btnSettings := menu.Text("⚙️ Settings")
	menu.Reply(
		menu.Row(btnWallet, btnGuardrails),
		menu.Row(btnAgents, btnSettings),
	)

	walletMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnBalance := walletMenu.Text("💰 Balances")
	btnAddress := walletMenu.Text("📍 Addresses")
	btnSend := walletMenu.Text("💸 Send")
	btnExport := walletMenu.Text("🔑 Export Keys")
	btnBack := walletMenu.Text("⬅️ Back")
	walletMenu.Reply(
		walletMenu.Row(btnBalance, btnAddress),
		walletMenu.Row(btnSend, btnExport),
		walletMenu.Row(btnBack),
	)

	guardrailMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnViewLimits := guardrailMenu.Text("📊 View Limits")
	btnSetDaily := guardrailMenu.Text("🚫 Set Daily Limit")
	btnSetSlippage := guardrailMenu.Text("📉 Set Slippage")
	guardrailMenu.Reply(
		guardrailMenu.Row(btnViewLimits),
		guardrailMenu.Row(btnSetDaily, btnSetSlippage),
		guardrailMenu.Row(btnBack),
	)

	agentMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnListAgents := agentMenu.Text("📋 Active Agents")
	btnBrowseAgents := agentMenu.Text("🔍 Browse Strategies")
	agentMenu.Reply(
		agentMenu.Row(btnListAgents, btnBrowseAgents),
		agentMenu.Row(btnBack),
	)

	settingsMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
	btnSwitch := settingsMenu.Text("🔄 Switch Account")
	btnRefresh := settingsMenu.Text("🔃 Refresh Session")
	settingsMenu.Reply(
		settingsMenu.Row(btnSwitch, btnRefresh),
		settingsMenu.Row(btnBack),
	)

	// --- Helper: Resolve Account ---	getAccountID := func(tgID int64) string {
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
			return c.Send("Welcome! You don't have an account yet. Send /setup to create one or link a local account using `settlerwallet link` from your CLI.", menu)
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
		userStates[c.Sender().ID] = &setupState{step: 1, action: "setup"}
		return c.Send("🔐 Enter a password to encrypt your vault:")
	})
	b.Handle(telebot.OnText, func(c telebot.Context) error {
		uid := c.Sender().ID
		state, ok := userStates[uid]
		if !ok {
			return nil
		}

		switch state.action {
		case "setup":
			if state.step == 1 {
				state.password = c.Text()
				state.step = 2
				c.Delete()
				return c.Send("✅ Password set. Send /confirm to generate your vault.")
			}
		case "limit":
			if state.step == 1 {
				limit := c.Text()
				accountID := getAccountID(uid)
				rules, _ := db.GetRules(accountID)
				if rules == nil {
					rules = &persistence.UserRules{AccountID: accountID, MaxSlippage: 1.0, CurrentSpend: "0", LastReset: time.Now().Unix()}
				}
				rules.DailyLimit = limit
				db.SaveRules(*rules)
				delete(userStates, uid)
				return c.Send(fmt.Sprintf("✅ Daily limit set to: `%s`", limit), telebot.ModeMarkdown)
			}
		case "send":
			switch state.step {
			case 1: // Selected Chain (handled via text button or typing)
				state.data["chain"] = c.Text()
				state.step = 2
				return c.Send("📍 Enter destination address:")
			case 2: // Destination Address
				state.data["to"] = c.Text()
				state.step = 3
				return c.Send("💰 Enter amount to send (in native units):")
			case 3: // Amount
				state.data["amount"] = c.Text()
				state.step = 4
				c.Delete()
				return c.Send("🔐 Enter vault password to sign transaction:")
			case 4: // Password & Execution
				state.password = c.Text()
				c.Delete()
				// TODO: Logic for signing and broadcasting
				delete(userStates, uid)
				return c.Send("⏳ Transaction processing... (Signing implementation in progress)")
			}
		}
		return nil
	})

	b.Handle(&btnSend, func(c telebot.Context) error {
		uid := c.Sender().ID
		userStates[uid] = &setupState{step: 1, action: "send", data: make(map[string]string)}
		
		chainMenu := &telebot.ReplyMarkup{ResizeKeyboard: true}
		chainMenu.Reply(chainMenu.Row(chainMenu.Text("BNB"), chainMenu.Text("Solana")))
		return c.Send("🚀 **New Transfer**\nSelect target chain:", chainMenu, telebot.ModeMarkdown)
	})

	b.Handle(&btnSetDaily, func(c telebot.Context) error {
		uid := c.Sender().ID
		userStates[uid] = &setupState{step: 1, action: "limit"}
		return c.Send("🚫 **Set Daily Spend Limit**\nEnter limit in wei/lamports:")
	})

	b.Handle(&btnExport, func(c telebot.Context) error {
		return c.Send("⚠️ **High Security Action**\nTo export your keys, run `settlerwallet export` from your secure CLI terminal. This action is restricted from the mobile bot for your safety.")
	})

	b.Handle(&btnListAgents, func(c telebot.Context) error {
		return c.Send("🤖 **Active Agents**\n0 active goroutines for this account.")
	})

	b.Handle(&btnBrowseAgents, func(c telebot.Context) error {
		return c.Send("🔍 **Available Strategies**\n\n1. `Auto-Compounder` - BSC Yield Aggregator (Coming Soon)\n2. `Stop-Loss` - DEX protection loop")
	})

	b.Handle(&btnRefresh, func(c telebot.Context) error {
		return c.Send("✅ Session refreshed.")
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
		return c.Send("💳 **Wallet Management**\nManage your accounts and assets.", walletMenu, telebot.ModeMarkdown)
	})

	b.Handle(&btnGuardrails, func(c telebot.Context) error {
		return c.Send("🛡️ **Guardrail Engine**\nRisk management and spend limits.", guardrailMenu, telebot.ModeMarkdown)
	})

	b.Handle(&btnAgents, func(c telebot.Context) error {
		return c.Send("🤖 **Strategy Agents**\nAutonomous financial goroutines.", agentMenu, telebot.ModeMarkdown)
	})

	b.Handle(&btnSettings, func(c telebot.Context) error {
		return c.Send("⚙️ **Settings**\nSystem configuration and account switching.", settingsMenu, telebot.ModeMarkdown)
	})

	b.Handle(&btnViewLimits, func(c telebot.Context) error {
		accountID := getAccountID(c.Sender().ID)
		rules, _ := db.GetRules(accountID)
		if rules == nil {
			return c.Send("ℹ️ No guardrails set for this account. Using defaults (1.0% slippage, 1 ETH/BNB limit).")
		}
		msg := fmt.Sprintf("📊 **Guardrail Status**\n\nDaily Limit: `%s` wei/lam\nCurrent Spend: `%s`\nMax Slippage: `%.1f%%`", 
			rules.DailyLimit, rules.CurrentSpend, rules.MaxSlippage)
		return c.Send(msg, telebot.ModeMarkdown)
	})

	b.Handle(&btnBack, func(c telebot.Context) error {
		return c.Send("🏠 **Main Menu**", menu, telebot.ModeMarkdown)
	})

	b.Handle(&btnSwitch, func(c telebot.Context) error {
		return c.Send("To switch accounts, please use the `/switch` command.")
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
}
