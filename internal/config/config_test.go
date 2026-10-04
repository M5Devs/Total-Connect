package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGetConfigPath_EnvOverride(t *testing.T) {
	customPath := "/tmp/custom_rclone.conf"
	os.Setenv("RCLONE_CONFIG", customPath)
	defer os.Unsetenv("RCLONE_CONFIG")

	path := GetConfigPath()
	if path != customPath {
		t.Errorf("expected %s, got %s", customPath, path)
	}
}

func TestGetConfigPath_Default(t *testing.T) {
	os.Unsetenv("RCLONE_CONFIG")

	path := GetConfigPath()
	if path == "" {
		t.Error("expected non-empty config path")
	}

	home, _ := os.UserHomeDir()
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		expected := filepath.Join(appData, "rclone", "rclone.conf")
		if appData != "" && path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	} else {
		expected := filepath.Join(home, ".config", "rclone", "rclone.conf")
		if path != expected {
			t.Errorf("expected %s, got %s", expected, path)
		}
	}
}

func TestConfigExists(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "rclone.conf")

	os.Setenv("RCLONE_CONFIG", configFile)
	defer os.Unsetenv("RCLONE_CONFIG")

	if ConfigExists() {
		t.Error("expected ConfigExists to return false for non-existent file")
	}

	err := os.WriteFile(configFile, []byte("[testremote]\ntype = local\n"), 0600)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	if !ConfigExists() {
		t.Error("expected ConfigExists to return true for existing file")
	}
}
