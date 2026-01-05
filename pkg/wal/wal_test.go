package wal

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWALEntry_ToJSON(t *testing.T) {
	entry := &WALEntry{
		ID:        "test-id",
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		LSN:       "0/1234567",
		Operation: OpInsert,
		Schema:    "public",
		Table:     "test_table",
		Data: map[string]interface{}{
			"id":   1,
			"name": "test",
		},
	}

	data, err := entry.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	if len(data) == 0 {
		t.Error("Expected non-empty JSON data")
	}
}

func TestFromJSON(t *testing.T) {
	jsonData := []byte(`{
		"id": "test-id",
		"timestamp": "2024-01-01T12:00:00Z",
		"lsn": "0/1234567",
		"operation": "INSERT",
		"schema": "public",
		"table": "test_table",
		"data": {"id": 1, "name": "test"}
	}`)

	entry, err := FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if entry.ID != "test-id" {
		t.Errorf("Expected ID test-id, got %s", entry.ID)
	}

	if entry.Operation != OpInsert {
		t.Errorf("Expected operation INSERT, got %s", entry.Operation)
	}

	if entry.Schema != "public" {
		t.Errorf("Expected schema public, got %s", entry.Schema)
	}
}

func TestLogWriter_WriteEntry(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewLogWriter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create log writer: %v", err)
	}
	defer writer.Close()

	entry := &WALEntry{
		ID:        "test-id",
		Timestamp: time.Now(),
		LSN:       "0/1234567",
		Operation: OpInsert,
		Schema:    "public",
		Table:     "test_table",
		Data: map[string]interface{}{
			"id": 1,
		},
	}

	err = writer.WriteEntry(entry)
	if err != nil {
		t.Fatalf("Failed to write entry: %v", err)
	}

	// Check that file was created
	files, err := filepath.Glob(filepath.Join(tmpDir, "wal_*.log"))
	if err != nil {
		t.Fatalf("Failed to list log files: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected at least one log file")
	}
}

func TestLogReader_ReadAll(t *testing.T) {
	tmpDir := t.TempDir()

	// Write some entries
	writer, err := NewLogWriter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create log writer: %v", err)
	}

	entries := []*WALEntry{
		{
			ID:        "test-1",
			Timestamp: time.Now(),
			LSN:       "0/1234567",
			Operation: OpInsert,
			Data:      map[string]interface{}{"id": 1},
		},
		{
			ID:        "test-2",
			Timestamp: time.Now(),
			LSN:       "0/1234568",
			Operation: OpUpdate,
			Data:      map[string]interface{}{"id": 2},
		},
	}

	for _, entry := range entries {
		if err := writer.WriteEntry(entry); err != nil {
			t.Fatalf("Failed to write entry: %v", err)
		}
	}
	writer.Close()

	// Read entries
	reader := NewLogReader(tmpDir)
	readEntries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read entries: %v", err)
	}

	if len(readEntries) != len(entries) {
		t.Errorf("Expected %d entries, got %d", len(entries), len(readEntries))
	}

	if len(readEntries) > 0 && readEntries[0].ID != "test-1" {
		t.Errorf("Expected first entry ID test-1, got %s", readEntries[0].ID)
	}
}

func TestLogReader_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	reader := NewLogReader(tmpDir)
	entries, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("Failed to read from empty directory: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("Expected 0 entries from empty directory, got %d", len(entries))
	}
}

func TestOperationTypes(t *testing.T) {
	tests := []struct {
		op       OperationType
		expected string
	}{
		{OpInsert, "INSERT"},
		{OpUpdate, "UPDATE"},
		{OpDelete, "DELETE"},
		{OpDDL, "DDL"},
	}

	for _, tt := range tests {
		t.Run(string(tt.op), func(t *testing.T) {
			if string(tt.op) != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, string(tt.op))
			}
		})
	}
}

func TestLogWriterClose(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewLogWriter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create log writer: %v", err)
	}

	// Should be able to close without error
	err = writer.Close()
	if err != nil {
		t.Errorf("Failed to close writer: %v", err)
	}
}

func TestLogWriterConcurrentWrites(t *testing.T) {
	tmpDir := t.TempDir()

	writer, err := NewLogWriter(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create log writer: %v", err)
	}
	defer writer.Close()

	// Write concurrently
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			entry := &WALEntry{
				ID:        fmt.Sprintf("test-%d", id),
				Timestamp: time.Now(),
				LSN:       "0/1234567",
				Operation: OpInsert,
				Data:      map[string]interface{}{"id": id},
			}
			writer.WriteEntry(entry)
			done <- true
		}(i)
	}

	// Wait for all writes
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify we can read the file
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("Failed to read directory: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected log files to be created")
	}
}
