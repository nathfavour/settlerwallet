package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath          string
	accountNameFlag string
)

var rootCmd = &cobra.Command{
	Use:   "settler",
	Short: "settlerWallet: Your agentic financial partner.",
	Long: `settlerWallet is a multi-chain, agentic wallet that helps you manage assets
and run strategies on BNB and Solana.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if accountNameFlag == "" {
			cfg := loadConfig()
			accountNameFlag = cfg.ActiveAccount
		}
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default is ~/.config/settlerwallet/settler.db)")
	rootCmd.PersistentFlags().StringVarP(&accountNameFlag, "name", "n", "", "Account name to use (default from config)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func main() {
	Execute()
}
