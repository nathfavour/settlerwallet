package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathfavour/settlerwallet/internal/blockchain"
	"github.com/nathfavour/settlerwallet/pkg/utils"
	"github.com/spf13/cobra"
)

// --- Styles ---
var (
	accentColor = lipgloss.Color("#7D56F4")
	greenColor  = lipgloss.Color("#00FF88")
	yellowColor = lipgloss.Color("#FFD700")
	subtleColor = lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"}

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(accentColor).
			Padding(0, 1).
			MarginBottom(1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(accentColor).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(accentColor).
			Padding(1, 2).
			MarginRight(2)

	balStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(greenColor)

	addrStyle = lipgloss.NewStyle().
			Foreground(subtleColor).
			Italic(true)

	statusStyle = lipgloss.NewStyle().
			Foreground(yellowColor)

	helpStyle = lipgloss.NewStyle().Foreground(subtleColor)
)

const asciiArt = `
  ____  _____ _____ _____ _      _____ ____  
 / ___|| ____|_   _|_   _| |    | ____|  _ \ 
 \___ \|  _|   | |   | | | |    |  _| | |_) |
  ___) | |___  | |   | | | |___ | |___|  _ < 
 |____/|_____| |_|   |_| |_____|_____|_| \_\
`

// --- Key Map ---
type keyMap struct {
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Help   key.Binding
	Quit   key.Binding
	Reload key.Binding
}

var keys = keyMap{
	Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "up")),
	Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "down")),
	Left:   key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "prev tab")),
	Right:  key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "next tab")),
	Help:   key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
	Quit:   key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Reload: key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "refresh balances")),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Left, k.Right, k.Reload, k.Help, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Reload, k.Help, k.Quit},
	}
}

// --- Model ---
type tokenData struct {
	Symbol  string
	Balance string
}

type walletData struct {
	Name    string
	Chain   string
	Address string
	Balance string
	Tokens  []tokenData
}

type settlerModel struct {
	activeTab  int
	tabs       []string
	wallets    []walletData
	account    string
	bnbClient  *blockchain.BNBClient
	solClient  *blockchain.SolanaClient
	loading    bool
	spinner    spinner.Model
	help       help.Model
	keys       keyMap
	width      int
	height     int
	lastUpdate time.Time
}

type balanceUpdateMsg []walletData

func initialModel() settlerModel {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(accentColor)

	cfg := loadConfig()
	account := cfg.ActiveAccount

	m := settlerModel{
		activeTab: 0,
		tabs:      []string{"Overview", "BNB Chain", "Solana", "Settings"},
		account:   account,
		spinner:   s,
		help:      help.New(),
		keys:      keys,
		loading:   true,
	}

	m.bnbClient, _ = blockchain.NewBNBClient("https://bsc-dataseed.binance.org")
	m.solClient, _ = blockchain.NewSolanaClient("https://api.mainnet-beta.solana.com")

	return m
}

func (m settlerModel) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.fetchData)
}

func (m settlerModel) fetchData() tea.Msg {
	db, err := initDB()
	if err != nil {
		return nil
	}
	defer db.Close()

	accountID := fmt.Sprintf("local:%s", m.account)
	wallets, _ := db.GetWallets(accountID)

	var results []walletData
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for _, w := range wallets {
		var balStr string
		var tokens []tokenData

		if w.Chain == "BNB" {
			b, err := m.bnbClient.GetBalance(ctx, w.Address)
			if err == nil {
				balStr = utils.FormatBalance(b.Amount, 18) + " BNB"
			}
			tbs, err := m.bnbClient.GetTokenBalances(ctx, w.Address)
			if err == nil {
				for _, tb := range tbs {
					tokens = append(tokens, tokenData{
						Symbol:  tb.Symbol,
						Balance: utils.FormatBalance(tb.Amount, tb.Decimals) + " " + tb.Symbol,
					})
				}
			}
		} else if w.Chain == "SOL" {
			b, err := m.solClient.GetBalance(ctx, w.Address)
			if err == nil {
				balStr = utils.FormatBalance(b.Amount, 9) + " SOL"
			}
			tbs, err := m.solClient.GetTokenBalances(ctx, w.Address)
			if err == nil {
				for _, tb := range tbs {
					tokens = append(tokens, tokenData{
						Symbol:  tb.Symbol,
						Balance: utils.FormatBalance(tb.Amount, tb.Decimals) + " " + tb.Symbol,
					})
				}
			}
		}

		results = append(results, walletData{
			Name:    w.Name,
			Chain:   w.Chain,
			Address: w.Address,
			Balance: balStr,
			Tokens:  tokens,
		})
	}
	return balanceUpdateMsg(results)
}

func (m settlerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.Right):
			m.activeTab = (m.activeTab + 1) % len(m.tabs)
		case key.Matches(msg, m.keys.Left):
			m.activeTab = (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
		case key.Matches(msg, m.keys.Reload):
			m.loading = true
			return m, m.fetchData
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case balanceUpdateMsg:
		m.wallets = msg
		m.loading = false
		m.lastUpdate = time.Now()
		return m, nil
	}
	return m, nil
}

func (m settlerModel) View() string {
	doc := strings.Builder{}

	// --- Header ---
	doc.WriteString(lipgloss.NewStyle().Foreground(accentColor).Render(asciiArt))
	doc.WriteString("\n")
	doc.WriteString(titleStyle.Render(fmt.Sprintf("SETTLER WALLET | ACCOUNT: %s", strings.ToUpper(m.account))))
	doc.WriteString("\n\n")

	// --- Tabs ---
	var renderedTabs []string
	for i, t := range m.tabs {
		style := lipgloss.NewStyle().Padding(0, 2)
		if i == m.activeTab {
			style = style.Bold(true).Border(lipgloss.NormalBorder(), false, false, true, false).BorderForeground(accentColor)
		} else {
			style = style.Foreground(subtleColor)
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}
	doc.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...))
	doc.WriteString("\n\n")

	// --- Body ---
	switch m.activeTab {
	case 0: // Overview
		doc.WriteString(m.renderOverview())
	case 1: // BNB
		doc.WriteString(m.renderChain("BNB"))
	case 2: // Solana
		doc.WriteString(m.renderChain("SOL"))
	case 3: // Settings
		doc.WriteString(headerStyle.Render("Configuration Settings"))
		doc.WriteString(fmt.Sprintf("Active Account: %s\n", m.account))
		doc.WriteString("Network: Mainnet (BNB, SOL)\n")
		doc.WriteString("Security: Local DB encrypted with AES-256 GCM\n")
	}

	// --- Footer ---
	footer := strings.Builder{}
	if m.loading {
		footer.WriteString(m.spinner.View() + " Refreshing balances...")
	} else {
		footer.WriteString(helpStyle.Render(fmt.Sprintf("Last Updated: %s", m.lastUpdate.Format("15:04:05"))))
	}
	footer.WriteString("\n\n")
	footer.WriteString(m.help.View(m.keys))

	renderedBody := doc.String()
	docHeight := lipgloss.Height(renderedBody)
	
	// Ensure footer is at bottom
	padding := m.height - docHeight - lipgloss.Height(footer.String()) - 2
	if padding > 0 {
		doc.WriteString(strings.Repeat("\n", padding))
	}
	doc.WriteString(footer.String())

	return lipgloss.NewStyle().Padding(1, 4).Render(doc.String())
}

func (m settlerModel) renderOverview() string {
	if len(m.wallets) == 0 {
		return "No wallets found. Use 'settler setup' to create one.\n"
	}

	var boxes []string
	for _, w := range m.wallets {
		content := strings.Builder{}
		content.WriteString(headerStyle.Render(w.Name) + "\n")
		content.WriteString(balStyle.Render(w.Balance) + "\n")
		content.WriteString(addrStyle.Render(w.Address[:12] + "..." + w.Address[len(w.Address)-8:]))
		
		if len(w.Tokens) > 0 {
			content.WriteString("\n" + subtleColor.Render("Tokens:") + "\n")
			for i, t := range w.Tokens {
				if i >= 3 {
					content.WriteString(subtleColor.Render(fmt.Sprintf("... +%d more", len(w.Tokens)-3)))
					break
				}
				content.WriteString(fmt.Sprintf("• %s\n", t.Balance))
			}
		}

		boxes = append(boxes, boxStyle.Render(content.String()))
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, boxes...)
}

func (m settlerModel) renderChain(chain string) string {
	found := false
	body := strings.Builder{}
	for _, w := range m.wallets {
		if w.Chain == chain {
			found = true
			body.WriteString(headerStyle.Render(fmt.Sprintf("Wallet: %s", w.Name)) + "\n")
			body.WriteString(fmt.Sprintf("Address: %s\n", w.Address))
			body.WriteString(fmt.Sprintf("Native: %s\n", balStyle.Render(w.Balance)))
			
			if len(w.Tokens) > 0 {
				body.WriteString("\n" + headerStyle.Render("Tokens:") + "\n")
				for _, t := range w.Tokens {
					body.WriteString(fmt.Sprintf("  - %s\n", t.Balance))
				}
			}
			body.WriteString("\n")
		}
	}
	if !found {
		return fmt.Sprintf("No %s wallets found for this account.\n", chain)
	}
	return body.String()
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Starts the beautiful settlerWallet TUI.",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error starting TUI: %v", err)
		}
	},
}
