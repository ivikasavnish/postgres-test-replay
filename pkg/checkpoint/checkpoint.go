package checkpoint

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ivikasavnish/postgres-test-replay/pkg/config"
	"github.com/ivikasavnish/postgres-test-replay/pkg/wal"
)

type Checkpoint struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	LSN         string    `json:"lsn"`
	EntryIndex  int       `json:"entry_index"`
	SessionID   string    `json:"session_id"`
}

type Manager struct {
	config      *config.Config
	checkpoints map[string]*Checkpoint
	mutex       sync.RWMutex
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:      cfg,
		checkpoints: make(map[string]*Checkpoint),
	}
}

func (m *Manager) CreateCheckpoint(name, description, lsn string, entryIndex int, sessionID string) (*Checkpoint, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	checkpoint := &Checkpoint{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		Timestamp:   time.Now(),
		LSN:         lsn,
		EntryIndex:  entryIndex,
		SessionID:   sessionID,
	}

	m.checkpoints[checkpoint.ID] = checkpoint

	if err := m.save(); err != nil {
		return nil, err
	}

	return checkpoint, nil
}

func (m *Manager) GetCheckpoint(id string) (*Checkpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	checkpoint, exists := m.checkpoints[id]
	if !exists {
		return nil, fmt.Errorf("checkpoint %s not found", id)
	}

	return checkpoint, nil
}

func (m *Manager) ListCheckpoints(sessionID string) ([]*Checkpoint, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	checkpoints := make([]*Checkpoint, 0)
	for _, cp := range m.checkpoints {
		if sessionID == "" || cp.SessionID == sessionID {
			checkpoints = append(checkpoints, cp)
		}
	}

	sort.Slice(checkpoints, func(i, j int) bool {
		return checkpoints[i].Timestamp.Before(checkpoints[j].Timestamp)
	})

	return checkpoints, nil
}

func (m *Manager) DeleteCheckpoint(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.checkpoints[id]; !exists {
		return fmt.Errorf("checkpoint %s not found", id)
	}

	delete(m.checkpoints, id)

	return m.save()
}

func (m *Manager) Load() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	checkpointPath := m.config.Storage.CheckpointPath
	if err := os.MkdirAll(checkpointPath, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	filename := filepath.Join(checkpointPath, "checkpoints.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open checkpoint file: %w", err)
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&m.checkpoints); err != nil {
		return fmt.Errorf("failed to decode checkpoints: %w", err)
	}

	return nil
}

func (m *Manager) save() error {
	checkpointPath := m.config.Storage.CheckpointPath
	if err := os.MkdirAll(checkpointPath, 0755); err != nil {
		return fmt.Errorf("failed to create checkpoint directory: %w", err)
	}

	filename := filepath.Join(checkpointPath, "checkpoints.json")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create checkpoint file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(m.checkpoints); err != nil {
		return fmt.Errorf("failed to encode checkpoints: %w", err)
	}

	return nil
}

type Navigator struct {
	walReader *wal.LogReader
	manager   *Manager
}

func NewNavigator(walReader *wal.LogReader, manager *Manager) *Navigator {
	return &Navigator{
		walReader: walReader,
		manager:   manager,
	}
}

func (n *Navigator) GetEntriesUpToCheckpoint(checkpointID string) ([]*wal.WALEntry, error) {
	checkpoint, err := n.manager.GetCheckpoint(checkpointID)
	if err != nil {
		return nil, err
	}

	allEntries, err := n.walReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read WAL entries: %w", err)
	}

	if checkpoint.EntryIndex >= len(allEntries) {
		return allEntries, nil
	}

	return allEntries[:checkpoint.EntryIndex+1], nil
}

func (n *Navigator) GetEntriesBetweenCheckpoints(startID, endID string) ([]*wal.WALEntry, error) {
	startCP, err := n.manager.GetCheckpoint(startID)
	if err != nil {
		return nil, err
	}

	endCP, err := n.manager.GetCheckpoint(endID)
	if err != nil {
		return nil, err
	}

	allEntries, err := n.walReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read WAL entries: %w", err)
	}

	startIdx := startCP.EntryIndex
	endIdx := endCP.EntryIndex

	if startIdx > endIdx {
		startIdx, endIdx = endIdx, startIdx
	}

	if endIdx >= len(allEntries) {
		endIdx = len(allEntries) - 1
	}

	return allEntries[startIdx : endIdx+1], nil
}
