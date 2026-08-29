// Package prefs stores Lamha's user preferences.
package prefs

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"sync"
)

const (
	DefaultMagnifierZoom = 5.0
	MinMagnifierZoom     = 2.0
	MaxMagnifierZoom     = 10.0
)

// Settings is the persisted preference file.
type Settings struct {
	MagnifierZoom float64 `json:"magnifier_zoom"`
}

// Store is the process-wide preference file.
type Store struct {
	mu      sync.RWMutex
	current Settings
	path    string
}

var (
	once sync.Once
	live *Store
)

// Load reads preferences from disk.
func Load() *Store {
	once.Do(func() {
		live = &Store{current: defaults(), path: configPath()}
		if data, err := os.ReadFile(live.path); err == nil {
			var raw Settings
			if json.Unmarshal(data, &raw) == nil {
				live.current.MagnifierZoom = ClampZoom(raw.MagnifierZoom)
			}
		}
	})
	return live
}

// Current returns the loaded preference store.
func Current() *Store {
	return Load()
}

// MagnifierZoom is the live lens magnification.
func (s *Store) MagnifierZoom() float64 {
	if s == nil {
		return DefaultMagnifierZoom
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ClampZoom(s.current.MagnifierZoom)
}

// SetMagnifierZoom writes and persists the lens magnification.
func (s *Store) SetMagnifierZoom(zoom float64) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.MagnifierZoom = ClampZoom(zoom)
	s.mu.Unlock()
	return s.save()
}

// ClampZoom keeps magnification inside the supported range.
func ClampZoom(zoom float64) float64 {
	if zoom == 0 {
		return DefaultMagnifierZoom
	}
	return math.Min(MaxMagnifierZoom, math.Max(MinMagnifierZoom, zoom))
}

func (s *Store) save() error {
	s.mu.RLock()
	raw := s.current
	path := s.path
	s.mu.RUnlock()

	data, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

func defaults() Settings {
	return Settings{MagnifierZoom: DefaultMagnifierZoom}
}

func configPath() string {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "lamha-settings.json"
		}
		config = filepath.Join(home, ".config")
	}
	return filepath.Join(config, "lamha", "settings.json")
}
