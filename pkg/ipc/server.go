package ipc

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/ivikasavnish/postgres-test-replay/pkg/checkpoint"
	"github.com/ivikasavnish/postgres-test-replay/pkg/config"
	"github.com/ivikasavnish/postgres-test-replay/pkg/session"
	"github.com/ivikasavnish/postgres-test-replay/pkg/wal"
)

type Server struct {
	config            *config.Config
	checkpointManager *checkpoint.Manager
	sessionManager    *session.Manager
	checkpointNav     *checkpoint.Navigator
	replayer          *session.Replayer
	server            *http.Server
}

func NewServer(cfg *config.Config, cpMgr *checkpoint.Manager, sessMgr *session.Manager, cpNav *checkpoint.Navigator, replayer *session.Replayer) *Server {
	return &Server{
		config:            cfg,
		checkpointManager: cpMgr,
		sessionManager:    sessMgr,
		checkpointNav:     cpNav,
		replayer:          replayer,
	}
}

func (s *Server) Start(addr string) error {
	mux := http.NewServeMux()

	// API endpoints
	mux.HandleFunc("/api/sessions", s.handleSessions)
	mux.HandleFunc("/api/sessions/", s.handleSession)
	mux.HandleFunc("/api/sessions/switch", s.handleSwitchSession)
	mux.HandleFunc("/api/checkpoints", s.handleCheckpoints)
	mux.HandleFunc("/api/checkpoints/", s.handleCheckpoint)
	mux.HandleFunc("/api/replay", s.handleReplay)
	mux.HandleFunc("/api/navigate", s.handleNavigate)
	mux.HandleFunc("/api/config", s.handleConfig)
	mux.HandleFunc("/api/wal-logs", s.handleWALLogs)
	mux.HandleFunc("/health", s.handleHealth)

	// Serve UI files
	if s.config.Server.UIPath != "" {
		fs := http.FileServer(http.Dir(s.config.Server.UIPath))
		mux.Handle("/", fs)
	}

	s.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to create listener: %w", err)
	}

	fmt.Printf("IPC Server listening on %s\n", addr)
	if s.config.Server.UIPath != "" {
		fmt.Printf("UI available at http://localhost%s\n", addr)
	}
	return s.server.Serve(listener)
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (s *Server) handleSessions(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessions, err := s.sessionManager.ListSessions()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(sessions)

	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Database    string `json:"database"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		sess, err := s.sessionManager.CreateSession(req.Name, req.Description, req.Database)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(sess)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	// Extract session ID from path
	sessionID := r.URL.Path[len("/api/sessions/"):]

	switch r.Method {
	case http.MethodGet:
		sess, err := s.sessionManager.GetSession(sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(sess)

	case http.MethodDelete:
		if err := s.sessionManager.DeleteSession(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleSwitchSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := s.sessionManager.SwitchSession(req.SessionID); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) handleCheckpoints(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		sessionID := r.URL.Query().Get("session_id")
		checkpoints, err := s.checkpointManager.ListCheckpoints(sessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(checkpoints)

	case http.MethodPost:
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			LSN         string `json:"lsn"`
			EntryIndex  int    `json:"entry_index"`
			SessionID   string `json:"session_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		cp, err := s.checkpointManager.CreateCheckpoint(req.Name, req.Description, req.LSN, req.EntryIndex, req.SessionID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := s.sessionManager.AddCheckpoint(req.SessionID, cp.ID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(cp)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCheckpoint(w http.ResponseWriter, r *http.Request) {
	checkpointID := r.URL.Path[len("/api/checkpoints/"):]

	switch r.Method {
	case http.MethodGet:
		cp, err := s.checkpointManager.GetCheckpoint(checkpointID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(cp)

	case http.MethodDelete:
		if err := s.checkpointManager.DeleteCheckpoint(checkpointID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleNavigate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CheckpointID string `json:"checkpoint_id"`
		StartID      string `json:"start_id"`
		EndID        string `json:"end_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var entries []*wal.WALEntry
	var err error

	if req.CheckpointID != "" {
		entries, err = s.checkpointNav.GetEntriesUpToCheckpoint(req.CheckpointID)
	} else if req.StartID != "" && req.EndID != "" {
		entries, err = s.checkpointNav.GetEntriesBetweenCheckpoints(req.StartID, req.EndID)
	} else {
		http.Error(w, "Invalid request: provide checkpoint_id or both start_id and end_id", http.StatusBadRequest)
		return
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries": entries,
		"count":   len(entries),
	})
}

func (s *Server) handleReplay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID    string `json:"session_id"`
		CheckpointID string `json:"checkpoint_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	sess, err := s.sessionManager.GetSession(req.SessionID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	entries, err := s.checkpointNav.GetEntriesUpToCheckpoint(req.CheckpointID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := s.replayer.ReplaySession(ctx, sess, entries); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":          "success",
		"entries_applied": len(entries),
	})
}

func (s *Server) handleConfig(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := map[string]interface{}{
		"input_dsn":  s.config.PrimaryDB.ToDSN(),
		"output_dsn": s.config.ReplicaDB.ToDSN(),
		"server": map[string]interface{}{
			"port":    s.config.Server.Port,
			"ui_path": s.config.Server.UIPath,
		},
		"storage": map[string]interface{}{
			"wal_log_path":     s.config.Storage.WALLogPath,
			"backup_path":      s.config.Storage.BackupPath,
			"session_path":     s.config.Storage.SessionPath,
			"checkpoint_path":  s.config.Storage.CheckpointPath,
		},
	}

	json.NewEncoder(w).Encode(response)
}

func (s *Server) handleWALLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	limitStr := r.URL.Query().Get("limit")
	limit := 100
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	walReader := wal.NewLogReader(s.config.Storage.WALLogPath)
	entries, err := walReader.ReadAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return the last N entries
	start := 0
	if len(entries) > limit {
		start = len(entries) - limit
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"entries":     entries[start:],
		"total_count": len(entries),
		"returned":    len(entries[start:]),
	})
}
