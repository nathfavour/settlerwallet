package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/nathfavour/settlerwallet/internal/blockchain"
	"github.com/nathfavour/settlerwallet/internal/vault"
	"github.com/nathfavour/settlerwallet/pkg/utils"
	"github.com/spf13/cobra"
)

var (
	subtle    = lipgloss.AdaptiveColor{Light: "#D9D9D9", Dark: "#383838"}
	highlight = lipgloss.AdaptiveColor{Light: "#874BFD", Dark: "#7D56F4"}
	special   = lipgloss.AdaptiveColor{Light: "#43BF6D", Dark: "#73F59F"}
	helpStyle = lipgloss.NewStyle().Foreground(lipgloss.AdaptiveColor{Light: "#9B9B9B", Dark: "#5C5C5C"})

	tab = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder(), true).
		BorderForeground(subtle).
		Padding(0, 1)

	activeTab = tab.Copy().
			BorderForeground(highlight)

	docStyle = lipgloss.NewStyle().Padding(1, 2, 1, 2)
)

func url(s string) string {
	return lipgloss.NewStyle().Foreground(special).Render(s)
}

type tickMsg time.Time

type tuiModel struct {
	Tabs       []string
	ActiveTab  int
	AddressBNB string
	AddressSOL string
	BalanceBNB string
	BalanceSOL string
	Mnemonic   string
	BNBClient  *blockchain.BNBClient
	SOLClient  *blockchain.SolanaClient
}

func initialTuiModel(mnemonic string) tuiModel {
	m := tuiModel{
		Tabs:     []string{"Dashboard", "BNB Chain", "Solana", "Settings"},
		Mnemonic: mnemonic,
	}

	if mnemonic != "" {
		seed := vault.GetSeedFromMnemonic(mnemonic)
		_, bnbAcc, _ := vault.DerivePrivateKey(seed, vault.ChainBNB, 0)
		_, solAcc, _ := vault.DerivePrivateKey(seed, vault.ChainSolana, 0)
		m.AddressBNB = bnbAcc
		m.AddressSOL = solAcc
	}

	m.BNBClient, _ = blockchain.NewBNBClient("https://bsc-dataseed.binance.org")
	m.SOLClient, _ = blockchain.NewSolanaClient("https://api.mainnet-beta.solana.com")

	m.BalanceBNB = "Loading..."
	m.BalanceSOL = "Loading..."

	return m
}

func (m tuiModel) Init() tea.Cmd {
	return tea.Batch(m.fetchBalances(), m.tick())
}

func (m tuiModel) tick() tea.Cmd {
	return tea.Every(time.Second*30, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m tuiModel) fetchBalances() tea.Cmd {
	return func() tea.Msg {
		if m.AddressBNB == "" || m.AddressSOL == "" {
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		bnbBal, err := m.BNBClient.GetBalance(ctx, m.AddressBNB)
		var bnbStr string
		if err != nil {
			bnbStr = "Error"
		} else {
			bnbStr = utils.FormatBalance(bnbBal.Amount, 18) + " BNB"
		}

		solBal, err := m.SOLClient.GetBalance(ctx, m.AddressSOL)
		var solStr string
		if err != nil {
			solStr = "Error"
		} else {
			solStr = utils.FormatBalance(solBal.Amount, 9) + " SOL"
		}

		return balanceMsg{BNB: bnbStr, SOL: solStr}
	}
}

type balanceMsg struct {
	BNB string
	SOL string
}

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "right", "l", "tab":
			m.ActiveTab = (m.ActiveTab + 1) % len(m.Tabs)
		case "left", "h", "shift+tab":
			m.ActiveTab = (m.ActiveTab - 1 + len(m.Tabs)) % len(m.Tabs)
		case "r":
			return m, m.fetchBalances()
		}
	case balanceMsg:
		m.BalanceBNB = msg.BNB
		m.BalanceSOL = msg.SOL
	case tickMsg:
		return m, m.fetchBalances()
	}
	return m, nil
}

func (m tuiModel) View() string {
	doc := strings.Builder{}

	var renderedTabs []string
	for i, t := range m.Tabs {
		var style lipgloss.Style
		isActive := i == m.ActiveTab
		if isActive {
			style = activeTab.Copy()
		} else {
			style = tab.Copy()
		}
		renderedTabs = append(renderedTabs, style.Render(t))
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)
	doc.WriteString(row)
	doc.WriteString("\n\n")

	switch m.ActiveTab {
	case 0:
		doc.WriteString(lipgloss.NewStyle().Bold(true).Render("Welcome to settlerWallet") + "\n\n")
		doc.WriteString(fmt.Sprintf("BNB Balance: %s\n", lipgloss.NewStyle().Foreground(highlight).Render(m.BalanceBNB)))
		doc.WriteString(fmt.Sprintf("SOL Balance: %s\n", lipgloss.NewStyle().Foreground(highlight).Render(m.BalanceSOL)))
	case 1:
		doc.WriteString(fmt.Sprintf("BNB Address: %s\n", m.AddressBNB))
		doc.WriteString(fmt.Sprintf("Balance: %s\n", m.BalanceBNB))
	case 2:
		doc.WriteString(fmt.Sprintf("Solana Address: %s\n", m.AddressSOL))
		doc.WriteString(fmt.Sprintf("Balance: %s\n", m.BalanceSOL))
	case 3:
		doc.WriteString("Settings\n\nRPC Nodes Connected.")
	}

	doc.WriteString("\n\n" + helpStyle.Render("q: quit • h/l: switch tabs • r: refresh"))
	return docStyle.Render(doc.String())
}

func init() {
	tuiCmd.Flags().StringVarP(&mnemonicInput, "mnemonic", "m", "", "BIP39 mnemonic for TUI session")
	rootCmd.AddCommand(tuiCmd)
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Starts the settlerWallet TUI.",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialTuiModel(mnemonicInput), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			log.Fatalf("Error starting TUI: %v", err)
		}
	},
}
