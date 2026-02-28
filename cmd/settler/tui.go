package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

type tuiModel struct {
	Tabs       []string
	ActiveTab  int
	AddressBNB string
	AddressSOL string
	BalanceBNB string
	BalanceSOL string
}

func initialTuiModel() tuiModel {
	return tuiModel{
		Tabs:       []string{"Dashboard", "BNB Chain", "Solana", "Settings"},
		ActiveTab:  0,
		AddressBNB: "0x742d35Cc6634C0532925a3b844Bc454e4438f44e", // Placeholder
		AddressSOL: "vines1vzrYbzRMRdu2thgeY4FG966QrTzAb9S56c3L2",  // Placeholder
		BalanceBNB: "1.245 BNB",
		BalanceSOL: "42.00 SOL",
	}
}

func (m tuiModel) Init() tea.Cmd {
	return nil
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
		}
	}
	return m, nil
}

func (m tuiModel) View() string {
	doc := strings.Builder{}

	// Tabs
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

	// Content based on active tab
	switch m.ActiveTab {
	case 0: // Dashboard
		doc.WriteString(lipgloss.NewStyle().Bold(true).Render("Welcome to settlerWallet") + "\n\n")
		doc.WriteString("Your multi-chain agentic partner is online.\n")
		doc.WriteString("Use the arrow keys or Tab to navigate.\n\n")
		doc.WriteString(fmt.Sprintf("BNB Balance: %s\n", lipgloss.NewStyle().Foreground(highlight).Render(m.BalanceBNB)))
		doc.WriteString(fmt.Sprintf("SOL Balance: %s\n", lipgloss.NewStyle().Foreground(highlight).Render(m.BalanceSOL)))
	case 1: // BNB
		doc.WriteString(lipgloss.NewStyle().Foreground(highlight).Render("BNB Smart Chain") + "\n\n")
		doc.WriteString(fmt.Sprintf("Address: %s\n", m.AddressBNB))
		doc.WriteString(fmt.Sprintf("Balance: %s\n", m.BalanceBNB))
	case 2: // Solana
		doc.WriteString(lipgloss.NewStyle().Foreground(highlight).Render("Solana") + "\n\n")
		doc.WriteString(fmt.Sprintf("Address: %s\n", m.AddressSOL))
		doc.WriteString(fmt.Sprintf("Balance: %s\n", m.BalanceSOL))
	case 3: // Settings
		doc.WriteString("Settings\n\n")
		doc.WriteString("RPC Nodes: " + url("https://bsc-dataseed.binance.org") + "\n")
		doc.WriteString("Daemon Status: " + lipgloss.NewStyle().Foreground(special).Render("Connected") + "\n")
	}

	doc.WriteString("\n\n" + helpStyle.Render("q: quit • h/l: switch tabs"))

	return docStyle.Render(doc.String())
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Starts the settlerWallet TUI.",
	Run: func(cmd *cobra.Command, args []string) {
		p := tea.NewProgram(initialTuiModel(), tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			fmt.Printf("Error starting TUI: %v", err)
			os.Exit(1)
		}
	},
}
