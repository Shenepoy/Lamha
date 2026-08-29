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

	ThemeSystem = "system"
	ThemeLight  = "light"
	ThemeDark   = "dark"

	LanguageSystem  = "system"
	LanguageEnglish = "en"
	LanguageArabic  = "ar"
)

// Settings is the persisted preference file.
type Settings struct {
	MagnifierZoom float64 `json:"magnifier_zoom"`
	Theme         string  `json:"theme"`
	Language      string  `json:"language"`
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
				live.current.Theme = NormalizeTheme(raw.Theme)
				live.current.Language = NormalizeLanguage(raw.Language)
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

// Theme is system, light, or dark.
func (s *Store) Theme() string {
	if s == nil {
		return ThemeSystem
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return NormalizeTheme(s.current.Theme)
}

// SetTheme writes and persists the color scheme.
func (s *Store) SetTheme(theme string) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.Theme = NormalizeTheme(theme)
	s.mu.Unlock()
	return s.save()
}

// Language is system, en, or ar.
func (s *Store) Language() string {
	if s == nil {
		return LanguageSystem
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return NormalizeLanguage(s.current.Language)
}

// SetLanguage writes and persists the UI language.
func (s *Store) SetLanguage(language string) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.Language = NormalizeLanguage(language)
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
	return Settings{
		MagnifierZoom: DefaultMagnifierZoom,
		Theme:         ThemeSystem,
		Language:      LanguageSystem,
	}
}

// NormalizeTheme accepts system, light, or dark.
func NormalizeTheme(theme string) string {
	switch theme {
	case ThemeLight, ThemeDark:
		return theme
	default:
		return ThemeSystem
	}
}

// NormalizeLanguage accepts system, en, or ar.
func NormalizeLanguage(language string) string {
	switch language {
	case LanguageEnglish, LanguageArabic:
		return language
	default:
		return LanguageSystem
	}
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
