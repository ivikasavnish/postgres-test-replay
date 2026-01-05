package config

import (
	"encoding/json"
	"os"
)

type Config struct {
	PrimaryDB   DatabaseConfig    `json:"primary_db"`
	ReplicaDB   DatabaseConfig    `json:"replica_db"`
	Storage     StorageConfig     `json:"storage"`
	Replication ReplicationConfig `json:"replication"`
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
