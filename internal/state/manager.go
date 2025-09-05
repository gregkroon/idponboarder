package state

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
	"time"

	"harness-onboarder/internal/models"
)

type Manager struct {
	filePath string
	state    models.State
	mutex    sync.RWMutex
}

func NewManager(filePath string) (*Manager, error) {
	manager := &Manager{
		filePath: filePath,
		state: models.State{
			ProcessedRepos: make(map[string]models.RepoState),
		},
	}

	if err := manager.loadState(); err != nil {
		log.Printf("Warning: failed to load state file, starting fresh: %v", err)
		manager.state.ProcessedRepos = make(map[string]models.RepoState)
	}

	return manager, nil
}

func (m *Manager) loadState() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, err := os.Stat(m.filePath); os.IsNotExist(err) {
		return nil
	}

	data, err := ioutil.ReadFile(m.filePath)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	if len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, &m.state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	if m.state.ProcessedRepos == nil {
		m.state.ProcessedRepos = make(map[string]models.RepoState)
	}

	log.Printf("Loaded state with %d processed repositories", len(m.state.ProcessedRepos))
	return nil
}

func (m *Manager) saveState() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	return m.saveStateUnsafe()
}

func (m *Manager) saveStateUnsafe() error {
	m.state.LastRun = time.Now()

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := ioutil.WriteFile(m.filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (m *Manager) ShouldSkip(repoFullName string, lastPushed time.Time) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	repoState, exists := m.state.ProcessedRepos[repoFullName]
	if !exists {
		return false
	}

	if repoState.Status == "error" {
		errorAge := time.Since(repoState.LastProcessed)
		if errorAge < 24*time.Hour {
			log.Printf("Skipping %s: recent error (%.1f hours ago)", 
				repoFullName, errorAge.Hours())
			return true
		}
		return false
	}

	if repoState.Status == "success" {
		if lastPushed.After(repoState.LastProcessed) {
			log.Printf("Repository %s has new changes since last processing", repoFullName)
			return false
		}
		
		successAge := time.Since(repoState.LastProcessed)
		if successAge < 7*24*time.Hour {
			log.Printf("Skipping %s: recently processed successfully (%.1f days ago)", 
				repoFullName, successAge.Hours()/24)
			return true
		}
	}

	return false
}

func (m *Manager) RecordSuccess(repoFullName string, lastPushed time.Time, mode string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.state.ProcessedRepos[repoFullName] = models.RepoState{
		LastProcessed: time.Now(),
		LastCommit:    "",
		Status:        "success",
	}

	log.Printf("Recorded successful processing of %s in %s mode", repoFullName, mode)
	
	if err := m.saveStateUnsafe(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}
}

func (m *Manager) RecordError(repoFullName string, err error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	errorMsg := err.Error()
	if len(errorMsg) > 500 {
		errorMsg = errorMsg[:500] + "..."
	}

	m.state.ProcessedRepos[repoFullName] = models.RepoState{
		LastProcessed: time.Now(),
		LastCommit:    "",
		Status:        "error",
		Error:         errorMsg,
	}

	log.Printf("Recorded error for %s: %s", repoFullName, errorMsg)
	
	if err := m.saveStateUnsafe(); err != nil {
		log.Printf("Warning: failed to save state: %v", err)
	}
}

func (m *Manager) RecordInProgress(repoFullName string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	existing := m.state.ProcessedRepos[repoFullName]
	existing.LastProcessed = time.Now()
	existing.Status = "in_progress"
	existing.Error = ""
	
	m.state.ProcessedRepos[repoFullName] = existing

	log.Printf("Marked %s as in progress", repoFullName)
}

func (m *Manager) GetStats() (int, int, int, time.Time) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	var successful, failed, inProgress int
	
	for _, state := range m.state.ProcessedRepos {
		switch state.Status {
		case "success":
			successful++
		case "error":
			failed++
		case "in_progress":
			inProgress++
		}
	}

	return successful, failed, inProgress, m.state.LastRun
}

func (m *Manager) GetRepoState(repoFullName string) (models.RepoState, bool) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	state, exists := m.state.ProcessedRepos[repoFullName]
	return state, exists
}

func (m *Manager) ListProcessedRepos() map[string]models.RepoState {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	result := make(map[string]models.RepoState)
	for k, v := range m.state.ProcessedRepos {
		result[k] = v
	}
	
	return result
}

func (m *Manager) CleanupOldEntries(maxAge time.Duration) int {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	cutoff := time.Now().Add(-maxAge)
	cleaned := 0

	for repoName, state := range m.state.ProcessedRepos {
		if state.LastProcessed.Before(cutoff) {
			delete(m.state.ProcessedRepos, repoName)
			cleaned++
		}
	}

	if cleaned > 0 {
		log.Printf("Cleaned up %d old state entries (older than %v)", cleaned, maxAge)
		if err := m.saveStateUnsafe(); err != nil {
			log.Printf("Warning: failed to save state after cleanup: %v", err)
		}
	}

	return cleaned
}

func (m *Manager) ResetRepoState(repoFullName string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.state.ProcessedRepos, repoFullName)
	log.Printf("Reset state for repository %s", repoFullName)
	
	if err := m.saveStateUnsafe(); err != nil {
		log.Printf("Warning: failed to save state after reset: %v", err)
	}
}

func (m *Manager) ResetAllState() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.state.ProcessedRepos = make(map[string]models.RepoState)
	log.Printf("Reset all repository state")
	
	if err := m.saveStateUnsafe(); err != nil {
		log.Printf("Warning: failed to save state after full reset: %v", err)
	}
}

func (m *Manager) ExportState(filePath string) error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state for export: %w", err)
	}

	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to export state: %w", err)
	}

	log.Printf("Exported state to %s", filePath)
	return nil
}

func (m *Manager) ImportState(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read import file: %w", err)
	}

	var importedState models.State
	if err := json.Unmarshal(data, &importedState); err != nil {
		return fmt.Errorf("failed to unmarshal imported state: %w", err)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if importedState.ProcessedRepos == nil {
		return fmt.Errorf("invalid state file: missing processed_repos")
	}

	m.state = importedState
	log.Printf("Imported state with %d repositories from %s", len(m.state.ProcessedRepos), filePath)
	
	return m.saveStateUnsafe()
}

func (m *Manager) Close() error {
	return m.saveState()
}