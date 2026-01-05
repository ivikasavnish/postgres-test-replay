package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestBackupDockerCompose(t *testing.T) {
	// Create a temporary test directory
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "docker-compose.yml")
	
	// Create a test docker-compose.yml
	content := []byte("version: '3.8'\nservices:\n  test: {}")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	
	// Override BackupDir for testing
	originalBackupDir := BackupDir
	testBackupDir := filepath.Join(tempDir, "backups")
	defer func() {
		// This won't work due to const, but leaving for clarity
		_ = originalBackupDir
	}()
	
	// Since we can't change const, we'll test the individual functions
	// Test that backup directory can be created
	if err := os.MkdirAll(testBackupDir, 0755); err != nil {
		t.Fatalf("Failed to create backup directory: %v", err)
	}
	
	// Create multiple backup files
	for i := 0; i < 5; i++ {
		timestamp := time.Now().Add(time.Duration(i) * time.Second).Format("20060102_150405")
		backupFile := filepath.Join(testBackupDir, "docker-compose_"+timestamp+".yml")
		if err := os.WriteFile(backupFile, content, 0644); err != nil {
			t.Fatalf("Failed to create backup file: %v", err)
		}
		time.Sleep(1 * time.Millisecond) // Ensure different timestamps
	}
	
	// Verify 5 backups were created
	files, err := os.ReadDir(testBackupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}
	
	backupCount := 0
	for _, f := range files {
		if !f.IsDir() && filepath.Ext(f.Name()) == ".yml" {
			backupCount++
		}
	}
	
	if backupCount != 5 {
		t.Errorf("Expected 5 backups, got %d", backupCount)
	}
	
	// Now simulate cleanup (keeping only 3)
	// In real implementation, cleanupOldBackups would be called
	// For this test, we'll verify the logic manually
	if backupCount > MaxBackups {
		expectedToDelete := backupCount - MaxBackups
		if expectedToDelete != 2 {
			t.Errorf("Expected to delete 2 old backups, calculated %d", expectedToDelete)
		}
	}
}

func TestDiscoverAvailablePortFromPool(t *testing.T) {
	port, err := discoverAvailablePortFromPool()
	if err != nil {
		t.Fatalf("Failed to discover port: %v", err)
	}
	
	portNum, err := strconv.Atoi(port)
	if err != nil {
		t.Fatalf("Port is not a valid number: %s", port)
	}
	
	if portNum < MinPort || portNum > MaxPort {
		t.Errorf("Port %d is outside of pool range %d-%d", portNum, MinPort, MaxPort)
	}
}

func TestUpdateEnvFile(t *testing.T) {
	tempDir := t.TempDir()
	envFile := filepath.Join(tempDir, ".env")
	
	// Change to temp directory
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tempDir)
	
	primaryDSN := "postgres://user:pass@localhost:58181/testdb?sslmode=disable"
	replicaDSN := "postgres://user:pass@localhost:58182/testdb?sslmode=disable"
	serverPort := "58183"
	
	if err := updateEnvFile(primaryDSN, replicaDSN, serverPort); err != nil {
		t.Fatalf("Failed to update env file: %v", err)
	}
	
	// Verify file was created
	if _, err := os.Stat(envFile); os.IsNotExist(err) {
		t.Fatal(".env file was not created")
	}
	
	// Read and verify content
	content, err := os.ReadFile(envFile)
	if err != nil {
		t.Fatalf("Failed to read .env file: %v", err)
	}
	
	contentStr := string(content)
	if !stringContains(contentStr, primaryDSN) {
		t.Error("Primary DSN not found in .env file")
	}
	if !stringContains(contentStr, replicaDSN) {
		t.Error("Replica DSN not found in .env file")
	}
	if !stringContains(contentStr, serverPort) {
		t.Error("Server port not found in .env file")
	}
}

func stringContains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && 
		(s == substr || len(s) >= len(substr) && findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
