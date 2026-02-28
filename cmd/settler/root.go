package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "settler",
	Short: "settlerWallet: Your agentic financial partner.",
	Long: `settlerWallet is a multi-chain, agentic wallet that helps you manage assets
and run strategies on BNB and Solana.`,
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
