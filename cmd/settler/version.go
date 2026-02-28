package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number of settlerWallet",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("settlerWallet v0.1.0-alpha")
	},
}
