package wal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type LogWriter struct {
	logPath     string
	currentFile *os.File
	writer      *bufio.Writer
	mutex       sync.Mutex
}

func NewLogWriter(logPath string) (*LogWriter, error) {
	if err := os.MkdirAll(logPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	lw := &LogWriter{
		logPath: logPath,
	}

	if err := lw.rotateLog(); err != nil {
		return nil, err
	}

	return lw, nil
}

func (lw *LogWriter) rotateLog() error {
	if lw.currentFile != nil {
		lw.writer.Flush()
		lw.currentFile.Close()
	}

	timestamp := time.Now().Format("20060102_150405")
	filename := filepath.Join(lw.logPath, fmt.Sprintf("wal_%s.log", timestamp))

	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to create log file: %w", err)
	}

	lw.currentFile = file
	lw.writer = bufio.NewWriter(file)

	return nil
}

func (lw *LogWriter) WriteEntry(entry *WALEntry) error {
	lw.mutex.Lock()
	defer lw.mutex.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal entry: %w", err)
	}

	if _, err := lw.writer.Write(data); err != nil {
		return fmt.Errorf("failed to write entry: %w", err)
	}

	if _, err := lw.writer.WriteString("\n"); err != nil {
		return fmt.Errorf("failed to write newline: %w", err)
	}

	return lw.writer.Flush()
}

func (lw *LogWriter) Close() error {
	lw.mutex.Lock()
	defer lw.mutex.Unlock()

	if lw.writer != nil {
		lw.writer.Flush()
	}

	if lw.currentFile != nil {
		return lw.currentFile.Close()
	}

	return nil
}

type LogReader struct {
	logPath string
}

func NewLogReader(logPath string) *LogReader {
	return &LogReader{
		logPath: logPath,
	}
}

func (lr *LogReader) ReadAll() ([]*WALEntry, error) {
	files, err := filepath.Glob(filepath.Join(lr.logPath, "wal_*.log"))
	if err != nil {
		return nil, fmt.Errorf("failed to list log files: %w", err)
	}

	entries := make([]*WALEntry, 0)

	for _, file := range files {
		fileEntries, err := lr.readFile(file)
		if err != nil {
			return nil, fmt.Errorf("failed to read file %s: %w", file, err)
		}
		entries = append(entries, fileEntries...)
	}

	return entries, nil
}

func (lr *LogReader) readFile(filename string) ([]*WALEntry, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	entries := make([]*WALEntry, 0)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		entry := &WALEntry{}
		if err := json.Unmarshal(line, entry); err != nil {
			continue
		}

		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return entries, nil
}
