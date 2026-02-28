package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	dbPath string
)

var rootCmd = &cobra.Command{
	Use:   "settler",
	Short: "settlerWallet: Your agentic financial partner.",
	Long: `settlerWallet is a multi-chain, agentic wallet that helps you manage assets
and run strategies on BNB and Solana.`,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&dbPath, "db", "", "Database path (default is ~/.config/settlerwallet/settler.db)")
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
