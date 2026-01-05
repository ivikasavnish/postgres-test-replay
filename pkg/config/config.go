package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	PrimaryDB   DatabaseConfig    `json:"primary_db"`
	ReplicaDB   DatabaseConfig    `json:"replica_db"`
	Storage     StorageConfig     `json:"storage"`
	Replication ReplicationConfig `json:"replication"`
	Server      ServerConfig      `json:"server"`
}

type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Database string `json:"database"`
	SSLMode  string `json:"ssl_mode"`
}

type StorageConfig struct {
	WALLogPath     string `json:"wal_log_path"`
	BackupPath     string `json:"backup_path"`
	SessionPath    string `json:"session_path"`
	CheckpointPath string `json:"checkpoint_path"`
}

type ReplicationConfig struct {
	SlotName        string `json:"slot_name"`
	PublicationName string `json:"publication_name"`
}

type ServerConfig struct {
	Port   int    `json:"port"`
	UIPath string `json:"ui_path"`
}

// ToDSN converts DatabaseConfig to DSN string
func (d *DatabaseConfig) ToDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		d.User, d.Password, d.Host, d.Port, d.Database, d.SSLMode)
}

// ParseDSN parses DSN string into DatabaseConfig
func ParseDSN(dsn string) (*DatabaseConfig, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid DSN: %w", err)
	}

	password, _ := u.User.Password()
	port := 5432
	if u.Port() != "" {
		port, _ = strconv.Atoi(u.Port())
	}

	query := u.Query()
	sslmode := query.Get("sslmode")
	if sslmode == "" {
		sslmode = "disable"
	}

	// Extract database name from path, handling empty path
	database := ""
	if len(u.Path) > 1 {
		database = u.Path[1:] // Remove leading /
	}

	return &DatabaseConfig{
		Host:     u.Hostname(),
		Port:     port,
		User:     u.User.Username(),
		Password: password,
		Database: database,
		SSLMode:  sslmode,
	}, nil
}

// LoadFromEnv loads configuration from .env file
func LoadFromEnv(envPath string) (*Config, error) {
	// Try to load .env file, but don't fail if it doesn't exist
	_ = godotenv.Load(envPath)

	cfg := &Config{}

	// Parse INPUT_DSN
	if inputDSN := os.Getenv("INPUT_DSN"); inputDSN != "" {
		primaryDB, err := ParseDSN(inputDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to parse INPUT_DSN: %w", err)
		}
		cfg.PrimaryDB = *primaryDB
	} else {
		return nil, fmt.Errorf("INPUT_DSN is required")
	}

	// Parse OUTPUT_DSN
	if outputDSN := os.Getenv("OUTPUT_DSN"); outputDSN != "" {
		replicaDB, err := ParseDSN(outputDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to parse OUTPUT_DSN: %w", err)
		}
		cfg.ReplicaDB = *replicaDB
	} else {
		return nil, fmt.Errorf("OUTPUT_DSN is required")
	}

	// Parse server configuration
	serverPort := 8080
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if p, err := strconv.Atoi(portStr); err == nil && p > 0 {
			serverPort = p
		} else {
			return nil, fmt.Errorf("invalid SERVER_PORT: %s", portStr)
		}
	}

	cfg.Server = ServerConfig{
		Port:   serverPort,
		UIPath: getEnvOrDefault("SERVER_UI_PATH", "./ui"),
	}

	// Storage configuration
	cfg.Storage = StorageConfig{
		WALLogPath:     getEnvOrDefault("WAL_LOG_PATH", "./waldata"),
		BackupPath:     getEnvOrDefault("BACKUP_PATH", "./backups"),
		SessionPath:    getEnvOrDefault("SESSION_PATH", "./sessions"),
		CheckpointPath: getEnvOrDefault("CHECKPOINT_PATH", "./checkpoints"),
	}

	// Replication configuration
	cfg.Replication = ReplicationConfig{
		SlotName:        getEnvOrDefault("REPLICATION_SLOT", "test_slot"),
		PublicationName: getEnvOrDefault("PUBLICATION_NAME", "test_publication"),
	}

	return cfg, nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func LoadConfig(path string) (*Config, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config := &Config{}
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(config); err != nil {
		return nil, err
	}

	return config, nil
}

func DefaultConfig() *Config {
	return &Config{
		PrimaryDB: DatabaseConfig{
			Host:     "localhost",
			Port:     5432,
			User:     "postgres",
			Password: "postgres",
			Database: "testdb",
			SSLMode:  "disable",
		},
		ReplicaDB: DatabaseConfig{
			Host:     "localhost",
			Port:     5433,
			User:     "postgres",
			Password: "postgres",
			Database: "testdb",
			SSLMode:  "disable",
		},
		Storage: StorageConfig{
			WALLogPath:     "./waldata",
			BackupPath:     "./backups",
			SessionPath:    "./sessions",
			CheckpointPath: "./checkpoints",
		},
		Replication: ReplicationConfig{
			SlotName:        "test_slot",
			PublicationName: "test_publication",
		},
		Server: ServerConfig{
			Port:   8080,
			UIPath: "./ui",
		},
	}
}

func (c *Config) Save(path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(c)
}
