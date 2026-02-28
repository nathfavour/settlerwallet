package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"

	"github.com/nathfavour/settlerwallet/internal/persistence"
	"golang.org/x/term"
)

type Config struct {
	ActiveAccount string `json:"active_account"`
}

func readPassword(prompt string) string {
	fmt.Print(prompt)
	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		log.Fatalf("❌ Error reading password: %v", err)
	}
	fmt.Println()
	return string(bytePassword)
}


func getAppDir() string {
	configDir, _ := os.UserConfigDir()
	return filepath.Join(configDir, "settlerwallet")
}

func getDBPath() string {
	return filepath.Join(getAppDir(), "settler.db")
}

func getPIDPath() string {
	return filepath.Join(getAppDir(), "settler.pid")
}

func getLogPath() string {
	return filepath.Join(getAppDir(), "settler.log")
}

func getConfigPath() string {
	return filepath.Join(getAppDir(), "config.json")
}

func initDB() (*persistence.DB, error) {
	appDir := getAppDir()
	os.MkdirAll(appDir, 0700)
	return persistence.NewDB(getDBPath())
}

func loadConfig() Config {
	var cfg Config
	data, err := os.ReadFile(getConfigPath())
	if err != nil {
		return cfg
	}
	json.Unmarshal(data, &cfg)
	return cfg
}

func saveConfig(cfg Config) error {
	appDir := getAppDir()
	os.MkdirAll(appDir, 0700)
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(getConfigPath(), data, 0600)
}
