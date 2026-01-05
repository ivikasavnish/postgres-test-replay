package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ivikasavnish/postgres-test-replay/pkg"
	"go.yaml.in/yaml/v2"
)

// Port pool configuration - reserved ports to avoid conflicts
const (
	MinPort    = 58180
	MaxPort    = 58190
	MaxBackups = 3
	BackupDir  = "./backups"
)

func main() {
	fmt.Println("=== PostgreSQL Test Replay - Docker Compose Setup ===")

	// Step 1: Backup existing docker-compose.yml
	composeFilePath := "./docker-compose.yml"
	if err := backupDockerCompose(composeFilePath); err != nil {
		fmt.Printf("Warning: Failed to backup docker-compose.yml: %v\n", err)
	}

	// Step 2: Read base docker compose file
	compose, err := ReadDockerComposeFile(composeFilePath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Docker Compose Version: %s\n", compose.Version)
	fmt.Printf("Postgres Primary Image: %s\n", compose.Services.PostgresPrimary.Image)
	fmt.Printf("Postgres Replica Image: %s\n", compose.Services.PostgresReplica.Image)

	// Step 3: Discover available ports from reserved pool
	primaryPort, err := discoverAvailablePortFromPool()
	if err != nil {
		panic(err)
	}
	replicaPort, err := discoverAvailablePortFromPool()
	if err != nil {
		panic(err)
	}
	serverPort, err := discoverAvailablePortFromPool()
	if err != nil {
		panic(err)
	}

	fmt.Printf("Discovered available Primary Port: %s\n", primaryPort)
	fmt.Printf("Discovered available Replica Port: %s\n", replicaPort)
	fmt.Printf("Discovered available Server Port: %s\n", serverPort)

	// Compose INPUT_DSN from docker-compose configuration
	INPUT_DSN := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s?sslmode=disable",
		compose.Services.PostgresPrimary.Environment.POSTGRESUSER,
		compose.Services.PostgresPrimary.Environment.POSTGRESPASSWORD,
		primaryPort,
		compose.Services.PostgresPrimary.Environment.POSTGRESDB,
	)

	fmt.Printf("Composed INPUT_DSN: %s\n", INPUT_DSN)

	component, err := url.Parse(INPUT_DSN)
	if err != nil {
		panic(err)
	}

	println("Scheme:", component.Scheme)
	println("User:", component.User.String())
	println("Password:", func() string {
		password, _ := component.User.Password()
		return password
	}())
	println("Host:", component.Hostname())
	println("Port:", component.Port())
	println("Path:", component.Path)
	println("RawQuery:", component.RawQuery)

	//  new designed DSN for replica - use same URL format as primary with all query params
	replicaDSNBase := fmt.Sprintf("postgres://%s:%s@localhost:%s/%s",
		component.User.Username(),
		func() string {
			password, _ := component.User.Password()
			return password
		}(),
		replicaPort,
		component.Path[1:], // remove leading '/'
	)
	// Append query parameters if they exist
	if component.RawQuery != "" {
		replicaDSNBase += "?" + component.RawQuery
	}
	designedDSN := replicaDSNBase
	println("Designed DSN:", designedDSN)

	// change postgres to version 18
	compose.Services.PostgresPrimary.Image = "postgres:18"
	compose.Services.PostgresReplica.Image = "postgres:18"
	// Print updated images
	fmt.Printf("Updated Postgres Primary Image: %s\n", compose.Services.PostgresPrimary.Image)
	fmt.Printf("Updated Postgres Replica Image: %s\n", compose.Services.PostgresReplica.Image)

	// Set discovered ports
	compose.Services.PostgresPrimary.Ports = []string{fmt.Sprintf("%s:5432", primaryPort)}
	compose.Services.PostgresReplica.Ports = []string{fmt.Sprintf("%s:5432", replicaPort)}

	// Print updated ports
	fmt.Printf("Updated Postgres Primary Ports: %v\n", compose.Services.PostgresPrimary.Ports)
	fmt.Printf("Updated Postgres Replica Ports: %v\n", compose.Services.PostgresReplica.Ports)
	// using os.WriteFile to write updated compose back to file

	//  update username and password
	compose.Services.PostgresPrimary.Environment.POSTGRESUSER = component.User.Username()
	compose.Services.PostgresPrimary.Environment.POSTGRESPASSWORD, _ = component.User.Password()
	compose.Services.PostgresReplica.Environment.POSTGRESUSER = component.User.Username()
	compose.Services.PostgresReplica.Environment.POSTGRESPASSWORD, _ = component.User.Password()

	//  change database name
	compose.Services.PostgresPrimary.Environment.POSTGRESDB = component.Path[1:]
	compose.Services.PostgresReplica.Environment.POSTGRESDB = component.Path[1:]

	// Print updated environment variables
	fmt.Printf("Updated Postgres Primary Environment: %+v\n", compose.Services.PostgresPrimary.Environment)
	fmt.Printf("Updated Postgres Replica Environment: %+v\n", compose.Services.PostgresReplica.Environment)

	// Create directories for persistent storage
	dirs := []string{
		"./data/postgres-primary",
		"./data/postgres-replica",
		"./wal/postgres-primary",
		"./wal/postgres-replica",
	}
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			panic(fmt.Sprintf("Failed to create directory %s: %v", dir, err))
		}
	}
	fmt.Println("Created persistent storage directories for data and WAL logs.")

	updatedData, err := yaml.Marshal(&compose)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(composeFilePath, updatedData, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("Updated docker-compose.yml written successfully.")
	//  update config.yaml file
	// create config.yaml file if not exists

	configFilePath := "./config.yaml"
	fileinfo, err := os.Stat(configFilePath)
	if os.IsNotExist(err) || fileinfo.Size() == 0 {
		_, err := os.Create(configFilePath)
		if err != nil {
			panic(err)
		}
		fmt.Println("Created new config.yaml file.")
	}
	configData := struct {
		PrimaryDSN string `yaml:"primary_dsn"`
		ReplicaDSN string `yaml:"replica_dsn"`
	}{
		PrimaryDSN: INPUT_DSN,
		ReplicaDSN: designedDSN,
	}
	configYAML, err := yaml.Marshal(&configData)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile(configFilePath, configYAML, 0644)
	if err != nil {
		panic(err)
	}
	fmt.Println("Updated config.yaml written successfully.")

	// Step 6: Create/Update .env file
	if err := updateEnvFile(INPUT_DSN, designedDSN, serverPort); err != nil {
		panic(err)
	}
	fmt.Println("Updated .env file successfully.")

	// Step 7: Stop any existing docker-compose services
	fmt.Println("\n=== Stopping existing Docker Compose services ===")
	if err := dockerComposeDown(); err != nil {
		fmt.Printf("Warning: Failed to stop existing services: %v\n", err)
	}

	// Step 8: Start docker-compose services
	fmt.Println("\n=== Starting Docker Compose services ===")
	if err := dockerComposeUp(); err != nil {
		panic(fmt.Errorf("Failed to start Docker Compose services: %v", err))
	}

	fmt.Println("\n=== Setup Complete ===")
	fmt.Printf("Primary DSN: %s\n", INPUT_DSN)
	fmt.Printf("Replica DSN: %s\n", designedDSN)
	fmt.Printf("Server will run on port: %s\n", serverPort)
	fmt.Println("\nYou can now run the application with:")
	fmt.Printf("  ./postgres-test-replay -mode ipc -addr :%s\n", serverPort)

}

func discoverAvailablePort() (string, error) {
	// Listen on a random port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer listener.Close()

	// Get the assigned port
	addr := listener.Addr().(*net.TCPAddr)
	return strconv.Itoa(addr.Port), nil
}

// discoverAvailablePortFromPool discovers an available port from the reserved pool
func discoverAvailablePortFromPool() (string, error) {
	for port := MinPort; port <= MaxPort; port++ {
		listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			// Port is in use, try next
			continue
		}
		listener.Close()
		return strconv.Itoa(port), nil
	}
	return "", fmt.Errorf("no available ports in pool %d-%d", MinPort, MaxPort)
}

// backupDockerCompose creates a timestamped backup of docker-compose.yml
func backupDockerCompose(filePath string) error {
	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // Nothing to backup
	}

	// Create backup directory
	if err := os.MkdirAll(BackupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Generate timestamped backup filename
	timestamp := time.Now().Format("20060102_150405")
	backupFile := filepath.Join(BackupDir, fmt.Sprintf("docker-compose_%s.yml", timestamp))

	// Copy file to backup
	source, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer source.Close()

	dest, err := os.Create(backupFile)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dest.Close()

	if _, err := io.Copy(dest, source); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	fmt.Printf("Created backup: %s\n", backupFile)

	// Clean up old backups
	if err := cleanupOldBackups(); err != nil {
		fmt.Printf("Warning: Failed to cleanup old backups: %v\n", err)
	}

	return nil
}

// cleanupOldBackups keeps only the most recent MaxBackups backups
func cleanupOldBackups() error {
	files, err := os.ReadDir(BackupDir)
	if err != nil {
		return err
	}

	// Filter only docker-compose backup files
	var backupFiles []os.DirEntry
	for _, file := range files {
		if !file.IsDir() && strings.HasPrefix(file.Name(), "docker-compose_") && strings.HasSuffix(file.Name(), ".yml") {
			backupFiles = append(backupFiles, file)
		}
	}

	// If we have more than MaxBackups, delete oldest ones
	if len(backupFiles) <= MaxBackups {
		return nil
	}

	// Sort by name (which includes timestamp)
	sort.Slice(backupFiles, func(i, j int) bool {
		return backupFiles[i].Name() < backupFiles[j].Name()
	})

	// Delete oldest files
	toDelete := len(backupFiles) - MaxBackups
	for i := 0; i < toDelete; i++ {
		filePath := filepath.Join(BackupDir, backupFiles[i].Name())
		if err := os.Remove(filePath); err != nil {
			fmt.Printf("Warning: Failed to remove old backup %s: %v\n", filePath, err)
		} else {
			fmt.Printf("Removed old backup: %s\n", filePath)
		}
	}

	return nil
}

// updateEnvFile creates or updates the .env file
func updateEnvFile(primaryDSN, replicaDSN, serverPort string) error {
	envContent := fmt.Sprintf(`# Database DSN Configuration
INPUT_DSN=%s
OUTPUT_DSN=%s

# Server Configuration
SERVER_PORT=%s
SERVER_UI_PATH=./ui

# Storage Configuration
WAL_LOG_PATH=./waldata
BACKUP_PATH=./backups
SESSION_PATH=./sessions
CHECKPOINT_PATH=./checkpoints

# Replication Configuration
REPLICATION_SLOT=test_slot
PUBLICATION_NAME=test_publication
`, primaryDSN, replicaDSN, serverPort)

	if err := os.WriteFile(".env", []byte(envContent), 0644); err != nil {
		return fmt.Errorf("failed to write .env file: %w", err)
	}

	return nil
}

// dockerComposeDown stops and removes docker-compose services
func dockerComposeDown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker-compose", "down", "-v")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose down failed: %w", err)
	}

	return nil
}

// dockerComposeUp starts docker-compose services
func dockerComposeUp() error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "docker-compose", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("docker-compose up failed: %w", err)
	}

	fmt.Println("Docker Compose services started successfully")

	// Wait a bit for services to be ready
	fmt.Println("Waiting for services to be ready...")
	time.Sleep(5 * time.Second)

	return nil
}

func ReadYAMLFile(filePath string, out interface{}) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, out)
}

func ReadDockerComposeFile(filePath string) (pkg.DockerCompose, error) {
	compose := pkg.DockerCompose{}
	err := ReadYAMLFile(filePath, &compose)
	if err != nil {
		return pkg.DockerCompose{}, err
	}
	return compose, nil
}
