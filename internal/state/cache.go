package state

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// CacheManager handles saving and loading the infrastructure state.
type CacheManager struct {
	cacheDir  string
	cacheFile string
}

// NewCacheManager creates a new cache manager for a given project path.
func NewCacheManager(projectPath string) *CacheManager {
	return &CacheManager{
		cacheDir:  filepath.Join(projectPath, ".cloudai"),
		cacheFile: filepath.Join(projectPath, ".cloudai", "cache.json"),
	}
}

// Save writes the given state to the cache file.
func (m *CacheManager) Save(state map[string]interface{}) error {
	if err := os.MkdirAll(m.cacheDir, 0755); err != nil {
		return err
	}

	bytes, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(m.cacheFile, bytes, 0644)
}

// Load reads the state from the cache file.
func (m *CacheManager) Load() (map[string]interface{}, error) {
	bytes, err := os.ReadFile(m.cacheFile)
	if err != nil {
		return nil, err
	}

	var state map[string]interface{}
	err = json.Unmarshal(bytes, &state)
	return state, err
}

// Exists checks if a cache file already exists.
func (m *CacheManager) Exists() bool {
	_, err := os.Stat(m.cacheFile)
	return err == nil
}
