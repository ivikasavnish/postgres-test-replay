package session

import (
	"context"
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

type Session struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Database    string    `json:"database"`
	Checkpoints []string  `json:"checkpoints"`
	Active      bool      `json:"active"`
}

type Manager struct {
	config   *config.Config
	sessions map[string]*Session
	active   string
	mutex    sync.RWMutex
}

func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config:   cfg,
		sessions: make(map[string]*Session),
	}
}

func (m *Manager) CreateSession(name, description, database string) (*Session, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session := &Session{
		ID:          uuid.New().String(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Database:    database,
		Checkpoints: make([]string, 0),
		Active:      false,
	}

	m.sessions[session.ID] = session

	if err := m.save(); err != nil {
		return nil, err
	}

	return session, nil
}

func (m *Manager) GetSession(id string) (*Session, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	session, exists := m.sessions[id]
	if !exists {
		return nil, fmt.Errorf("session %s not found", id)
	}

	return session, nil
}

func (m *Manager) ListSessions() ([]*Session, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].CreatedAt.After(sessions[j].CreatedAt)
	})

	return sessions, nil
}

func (m *Manager) SwitchSession(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, exists := m.sessions[id]
	if !exists {
		return fmt.Errorf("session %s not found", id)
	}

	for _, s := range m.sessions {
		s.Active = false
	}

	session.Active = true
	m.active = id

	return m.save()
}

func (m *Manager) GetActiveSession() (*Session, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if m.active == "" {
		return nil, fmt.Errorf("no active session")
	}

	return m.sessions[m.active], nil
}

func (m *Manager) AddCheckpoint(sessionID, checkpointID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Checkpoints = append(session.Checkpoints, checkpointID)
	session.UpdatedAt = time.Now()

	return m.save()
}

func (m *Manager) DeleteSession(id string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.sessions[id]; !exists {
		return fmt.Errorf("session %s not found", id)
	}

	delete(m.sessions, id)

	if m.active == id {
		m.active = ""
	}

	return m.save()
}

func (m *Manager) Load() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	sessionPath := m.config.Storage.SessionPath
	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	filename := filepath.Join(sessionPath, "sessions.json")
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return nil
	}

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open session file: %w", err)
	}
	defer file.Close()

	type SaveData struct {
		Sessions map[string]*Session `json:"sessions"`
		Active   string              `json:"active"`
	}

	var data SaveData
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&data); err != nil {
		return fmt.Errorf("failed to decode sessions: %w", err)
	}

	m.sessions = data.Sessions
	m.active = data.Active

	return nil
}

func (m *Manager) save() error {
	sessionPath := m.config.Storage.SessionPath
	if err := os.MkdirAll(sessionPath, 0755); err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}

	filename := filepath.Join(sessionPath, "sessions.json")
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create session file: %w", err)
	}
	defer file.Close()

	type SaveData struct {
		Sessions map[string]*Session `json:"sessions"`
		Active   string              `json:"active"`
	}

	data := SaveData{
		Sessions: m.sessions,
		Active:   m.active,
	}

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("failed to encode sessions: %w", err)
	}

	return nil
}

type Replayer struct {
	config *config.Config
}

func NewReplayer(cfg *config.Config) *Replayer {
	return &Replayer{
		config: cfg,
	}
}

func (r *Replayer) ReplaySession(ctx context.Context, session *Session, entries []*wal.WALEntry) error {
	// This would connect to the replica database and replay the WAL entries
	// For now, we'll just log that we would replay
	fmt.Printf("Replaying session %s with %d entries\n", session.ID, len(entries))

	for _, entry := range entries {
		if err := r.applyEntry(ctx, entry); err != nil {
			return fmt.Errorf("failed to apply entry %s: %w", entry.ID, err)
		}
	}

	return nil
}

func (r *Replayer) applyEntry(ctx context.Context, entry *wal.WALEntry) error {
	// Apply the WAL entry to the database
	// This is a placeholder - actual implementation would execute SQL
	fmt.Printf("Applying %s operation on %s.%s\n", entry.Operation, entry.Schema, entry.Table)
	return nil
}
