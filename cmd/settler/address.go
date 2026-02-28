package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(addressCmd)
}

var addressCmd = &cobra.Command{
	Use:   "address",
	Short: "Show your wallet addresses (placeholder).",
	Run: func(cmd *cobra.Command, args []string) {
		// Placeholder since we don't have a persistent vault yet in the CLI.
		fmt.Println("BNB Address:  0x742d35Cc6634C0532925a3b844Bc454e4438f44e")
		fmt.Println("Solana Address: vines1vzrYbzRMRdu2thgeY4FG966QrTzAb9S56c3L2")
	},
}
