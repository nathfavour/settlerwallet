package main

import (
	"fmt"
	"log"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(renameCmd)
}

var renameCmd = &cobra.Command{
	Use:   "rename <old_name> <new_name>",
	Short: "Rename a local account.",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		oldName := args[0]
		newName := args[1]

		db, err := initDB()
		if err != nil {
			log.Fatalf("❌ Database error: %v", err)
		}
		defer db.Close()

		oldID := fmt.Sprintf("local:%s", oldName)
		newID := fmt.Sprintf("local:%s", newName)

		// Check if old account exists
		acc, _ := db.GetAccount(oldID)
		if acc == nil {
			log.Fatalf("❌ Error: Account '%s' not found.", oldName)
		}

		// Check if new name is already taken
		exists, _ := db.GetAccount(newID)
		if exists != nil {
			log.Fatalf("❌ Error: Account name '%s' is already in use.", newName)
		}

		if err := db.RenameAccount(oldID, newID); err != nil {
			log.Fatalf("❌ Error renaming account: %v", err)
		}

		fmt.Printf("✅ Success! Account '%s' has been renamed to '%s'.\n", oldName, newName)
	},
}
