package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// State tracks the migration state
type State struct {
	ProcessedPages map[string]bool `json:"processed_pages"`
	mu             sync.RWMutex
}

// NewState creates a new empty state
func NewState() *State {
	return &State{
		ProcessedPages: make(map[string]bool),
	}
}

// LoadState loads the state from the state file
func LoadState() (*State, error) {
	statePath, err := GetStatePath()
	if err != nil {
		return nil, err
	}

	// If state file doesn't exist, return empty state
	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return NewState(), nil
	}

	data, err := os.ReadFile(statePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read state file: %w", err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if state.ProcessedPages == nil {
		state.ProcessedPages = make(map[string]bool)
	}

	return &state, nil
}

// SaveState saves the state to the state file
func (s *State) SaveState() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	statePath, err := GetStatePath()
	if err != nil {
		return err
	}

	// Ensure the directory exists
	stateDir := filepath.Dir(statePath)
	if err := os.MkdirAll(stateDir, 0755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// MarkProcessed marks a page as processed
func (s *State) MarkProcessed(pageID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ProcessedPages[pageID] = true
}

// IsProcessed checks if a page has been processed
func (s *State) IsProcessed(pageID string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.ProcessedPages[pageID]
}

// ClearState resets the state
func (s *State) ClearState() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ProcessedPages = make(map[string]bool)
}

// GetStatePath returns the state file path
func GetStatePath() (string, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "state.json"), nil
}

// ClearStateFile removes the state file from disk
func ClearStateFile() error {
	statePath, err := GetStatePath()
	if err != nil {
		return err
	}

	if _, err := os.Stat(statePath); os.IsNotExist(err) {
		return nil // Already doesn't exist
	}

	if err := os.Remove(statePath); err != nil {
		return fmt.Errorf("failed to remove state file: %w", err)
	}

	return nil
}
