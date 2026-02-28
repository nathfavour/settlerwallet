package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath          string
	accountNameFlag string
	killFlag        bool
)

var rootCmd = &cobra.Command{
	Use:   "settlerwallet",
	Short: "settlerWallet: Your agentic financial partner.",
	Long: `settlerWallet is a multi-chain, agentic wallet that helps you manage assets
and run strategies on BNB and Solana.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if accountNameFlag == "" {
			cfg := loadConfig()
			accountNameFlag = cfg.ActiveAccount
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		// If no subcommand is provided, handle as daemon start/kill
		if killFlag {
			daemonStopCmd.Run(cmd, args)
			return
		}
		daemonStartCmd.Run(cmd, args)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default is ~/.config/settlerwallet/settler.db)")
	rootCmd.PersistentFlags().StringVarP(&accountNameFlag, "name", "n", "", "Account name to use (default from config)")
	rootCmd.Flags().BoolVarP(&killFlag, "kill", "k", false, "Kill the running background daemon")
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
