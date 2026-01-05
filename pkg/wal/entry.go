package wal

import (
	"encoding/json"
	"time"
)

type OperationType string

const (
	OpInsert OperationType = "INSERT"
	OpUpdate OperationType = "UPDATE"
	OpDelete OperationType = "DELETE"
	OpDDL    OperationType = "DDL"
)

type WALEntry struct {
	ID           string                 `json:"id"`
	Timestamp    time.Time              `json:"timestamp"`
	LSN          string                 `json:"lsn"`
	Operation    OperationType          `json:"operation"`
	Schema       string                 `json:"schema"`
	Table        string                 `json:"table"`
	Data         map[string]interface{} `json:"data"`
	OldData      map[string]interface{} `json:"old_data,omitempty"`
	SQL          string                 `json:"sql,omitempty"`
	CheckpointID string                 `json:"checkpoint_id,omitempty"`
}

func (w *WALEntry) ToJSON() ([]byte, error) {
	return json.Marshal(w)
}

func FromJSON(data []byte) (*WALEntry, error) {
	entry := &WALEntry{}
	err := json.Unmarshal(data, entry)
	return entry, err
}
