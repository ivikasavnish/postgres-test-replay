package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ivikasavnish/postgres-test-replay/pkg/backup"
	"github.com/ivikasavnish/postgres-test-replay/pkg/checkpoint"
	"github.com/ivikasavnish/postgres-test-replay/pkg/config"
	"github.com/ivikasavnish/postgres-test-replay/pkg/ipc"
	"github.com/ivikasavnish/postgres-test-replay/pkg/replication"
	"github.com/ivikasavnish/postgres-test-replay/pkg/session"
	"github.com/ivikasavnish/postgres-test-replay/pkg/wal"
)

func main() {
	var (
		envPath    = flag.String("env", ".env", "Path to .env file")
		configPath = flag.String("config", "", "Path to configuration file (optional, overrides .env)")
		mode       = flag.String("mode", "listener", "Mode: listener, ipc, backup, restore")
		addr       = flag.String("addr", "", "IPC server address (optional, overrides config)")
		backupName = flag.String("backup", "", "Backup file name for restore mode")
		targetDB   = flag.String("target-db", "", "Target database for restore")
	)
	flag.Parse()

	cfg, err := loadConfig(*envPath, *configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	switch *mode {
	case "listener":
		runListener(cfg)
	case "ipc":
		serverAddr := *addr
		if serverAddr == "" {
			serverAddr = fmt.Sprintf(":%d", cfg.Server.Port)
		}
		runIPC(cfg, serverAddr)
	case "backup":
		runBackup(cfg)
	case "restore":
		if *backupName == "" || *targetDB == "" {
			log.Fatal("backup and target-db flags are required for restore mode")
		}
		runRestore(cfg, *backupName, *targetDB)
	default:
		log.Fatalf("Unknown mode: %s", *mode)
	}
}

func loadConfig(envPath, configPath string) (*config.Config, error) {
	// Try .env first
	if envPath != "" {
		cfg, err := config.LoadFromEnv(envPath)
		if err == nil {
			log.Printf("Loaded configuration from %s", envPath)
			return cfg, nil
		}
		// Check if it's a file not found error
		if os.IsNotExist(err) {
			log.Printf("Note: .env file not found at %s, will try other config sources", envPath)
		} else {
			log.Printf("Warning: Failed to parse .env file: %v", err)
		}
	}

	// Fall back to JSON config or default
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			cfg, err := config.LoadConfig(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config from %s: %w", configPath, err)
			}
			log.Printf("Loaded configuration from %s", configPath)
			return cfg, nil
		}
	}

	// Use default config
	cfg := config.DefaultConfig()
	log.Println("Using default configuration")
	return cfg, nil
}

func runListener(cfg *config.Config) {
	log.Println("Starting WAL replication listener...")

	walWriter, err := wal.NewLogWriter(cfg.Storage.WALLogPath)
	if err != nil {
		log.Fatalf("Failed to create WAL writer: %v", err)
	}
	defer walWriter.Close()

	listener := replication.NewListener(cfg, walWriter)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := listener.Connect(ctx); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer listener.Close()

	log.Println("Creating replication slot...")
	if err := listener.CreateReplicationSlot(ctx); err != nil {
		log.Printf("Warning: Failed to create replication slot (may already exist): %v", err)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	log.Println("Starting replication stream...")
	if err := listener.Start(ctx); err != nil && err != context.Canceled {
		log.Fatalf("Replication failed: %v", err)
	}

	log.Println("Listener stopped")
}

func runIPC(cfg *config.Config, addr string) {
	log.Printf("Starting IPC server on %s...", addr)

	checkpointMgr := checkpoint.NewManager(cfg)
	if err := checkpointMgr.Load(); err != nil {
		log.Fatalf("Failed to load checkpoints: %v", err)
	}

	sessionMgr := session.NewManager(cfg)
	if err := sessionMgr.Load(); err != nil {
		log.Fatalf("Failed to load sessions: %v", err)
	}

	walReader := wal.NewLogReader(cfg.Storage.WALLogPath)
	checkpointNav := checkpoint.NewNavigator(walReader, checkpointMgr)
	replayer := session.NewReplayer(cfg)

	server := ipc.NewServer(cfg, checkpointMgr, sessionMgr, checkpointNav, replayer)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("Error shutting down server: %v", err)
		}
	}()

	if err := server.Start(addr); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed: %v", err)
	}

	log.Println("IPC server stopped")
}

func runBackup(cfg *config.Config) {
	log.Println("Creating database backup...")

	backupMgr := backup.NewBackupManager(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	backupFile, err := backupMgr.CreateBackup(ctx, cfg.PrimaryDB.Database)
	if err != nil {
		log.Fatalf("Backup failed: %v", err)
	}

	log.Printf("Backup created successfully: %s", backupFile)
}

func runRestore(cfg *config.Config, backupFile, targetDB string) {
	log.Printf("Restoring backup %s to database %s...", backupFile, targetDB)

	backupMgr := backup.NewBackupManager(cfg)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := backupMgr.RestoreBackup(ctx, backupFile, targetDB); err != nil {
		log.Fatalf("Restore failed: %v", err)
	}

	log.Println("Restore completed successfully")
}
