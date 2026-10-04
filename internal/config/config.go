package config

import (
	"os"
	"path/filepath"
	"runtime"
)

// GetConfigPath returns the path to the rclone config file.
// It checks standard system locations (Windows: %APPDATA%/rclone/rclone.conf,
// Linux/macOS: ~/.config/rclone/rclone.conf). If RCLONE_CONFIG env var is set,
// that takes precedence.
func GetConfigPath() string {
	if envPath := os.Getenv("RCLONE_CONFIG"); envPath != "" {
		return envPath
	}

	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData != "" {
			return filepath.Join(appData, "rclone", "rclone.conf")
		}
		return filepath.Join(home, "AppData", "Roaming", "rclone", "rclone.conf")
	}

	return filepath.Join(home, ".config", "rclone", "rclone.conf")
}

// ConfigExists returns whether an rclone config file exists at the default/configured path.
func ConfigExists() bool {
	path := GetConfigPath()
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}
