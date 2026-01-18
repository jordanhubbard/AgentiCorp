package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the arbiter configuration
type Config struct {
	DatabasePath string
	KeyStorePath string
	DataDir      string
}

// Default returns the default configuration
func Default() (*Config, error) {
	// Use XDG_DATA_HOME or ~/.local/share as base directory
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get user home directory: %w", err)
		}
		dataHome = filepath.Join(home, ".local", "share")
	}

	dataDir := filepath.Join(dataHome, "arbiter")

	// Ensure data directory exists
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &Config{
		DatabasePath: filepath.Join(dataDir, "arbiter.db"),
		KeyStorePath: filepath.Join(dataDir, "keystore.json"),
		DataDir:      dataDir,
	}, nil
}

// GetPassword retrieves the unlock password from environment or prompts the user
// Environment variable takes precedence: ARBITER_PASSWORD
func GetPassword() (string, error) {
	// Check environment variable first
	if password := os.Getenv("ARBITER_PASSWORD"); password != "" {
		return password, nil
	}

	// Prompt user for password
	fmt.Print("Enter password to unlock key store: ")
	var password string
	_, err := fmt.Scanln(&password)
	if err != nil {
		return "", fmt.Errorf("failed to read password: %w", err)
	}

	if password == "" {
		return "", fmt.Errorf("password cannot be empty")
	}

	return password, nil
}
