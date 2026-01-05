package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.PrimaryDB.Host != "localhost" {
		t.Errorf("Expected primary host localhost, got %s", cfg.PrimaryDB.Host)
	}

	if cfg.PrimaryDB.Port != 5432 {
		t.Errorf("Expected primary port 5432, got %d", cfg.PrimaryDB.Port)
	}

	if cfg.Storage.WALLogPath != "./waldata" {
		t.Errorf("Expected WAL log path ./waldata, got %s", cfg.Storage.WALLogPath)
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.json")

	// Create and save config
	cfg := DefaultConfig()
	cfg.PrimaryDB.Host = "testhost"
	cfg.PrimaryDB.Port = 9999

	err := cfg.Save(configPath)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load config
	loadedCfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if loadedCfg.PrimaryDB.Host != "testhost" {
		t.Errorf("Expected host testhost, got %s", loadedCfg.PrimaryDB.Host)
	}

	if loadedCfg.PrimaryDB.Port != 9999 {
		t.Errorf("Expected port 9999, got %d", loadedCfg.PrimaryDB.Port)
	}
}

func TestLoadNonExistentConfig(t *testing.T) {
	_, err := LoadConfig("/non/existent/config.json")
	if err == nil {
		t.Error("Expected error loading non-existent config")
	}
}

func TestConfigValidation(t *testing.T) {
	cfg := DefaultConfig()

	// Test that required fields are set
	if cfg.PrimaryDB.Database == "" {
		t.Error("Primary database name should not be empty")
	}

	if cfg.Replication.SlotName == "" {
		t.Error("Replication slot name should not be empty")
	}

	if cfg.Replication.PublicationName == "" {
		t.Error("Publication name should not be empty")
	}
}

func TestConfigSaveCreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "subdir", "config.json")

	// This should fail because parent directory doesn't exist
	cfg := DefaultConfig()
	err := cfg.Save(configPath)
	if err == nil {
		t.Error("Expected error saving to non-existent directory")
	}

	// Create directory and try again
	os.MkdirAll(filepath.Dir(configPath), 0755)
	err = cfg.Save(configPath)
	if err != nil {
		t.Errorf("Failed to save config after creating directory: %v", err)
	}
}
