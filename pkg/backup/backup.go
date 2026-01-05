package backup

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/ivikasavnish/postgres-test-replay/pkg/config"
)

type BackupManager struct {
	config *config.Config
}

func NewBackupManager(cfg *config.Config) *BackupManager {
	return &BackupManager{
		config: cfg,
	}
}

func (bm *BackupManager) CreateBackup(ctx context.Context, dbName string) (string, error) {
	// Ensure backup directory exists
	if err := os.MkdirAll(bm.config.Storage.BackupPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate backup filename with timestamp
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(bm.config.Storage.BackupPath, fmt.Sprintf("%s_%s.sql", dbName, timestamp))

	// Build pg_dump command
	dbConfig := bm.config.PrimaryDB
	cmd := exec.CommandContext(ctx, "pg_dump",
		"-h", dbConfig.Host,
		"-p", fmt.Sprintf("%d", dbConfig.Port),
		"-U", dbConfig.User,
		"-d", dbConfig.Database,
		"-F", "c",
		"-f", backupFile,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("pg_dump failed: %w, output: %s", err, string(output))
	}

	return backupFile, nil
}

func (bm *BackupManager) RestoreBackup(ctx context.Context, backupFile string, targetDB string) error {
	dbConfig := bm.config.ReplicaDB
	cmd := exec.CommandContext(ctx, "pg_restore",
		"-h", dbConfig.Host,
		"-p", fmt.Sprintf("%d", dbConfig.Port),
		"-U", dbConfig.User,
		"-d", targetDB,
		"-c",
		backupFile,
	)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PGPASSWORD=%s", dbConfig.Password))

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pg_restore failed: %w, output: %s", err, string(output))
	}

	return nil
}

func (bm *BackupManager) ListBackups() ([]string, error) {
	backupDir := bm.config.Storage.BackupPath

	if _, err := os.Stat(backupDir); os.IsNotExist(err) {
		return []string{}, nil
	}

	files, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]string, 0)
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".sql" {
			backups = append(backups, file.Name())
		}
	}

	return backups, nil
}
