package keys

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/lamha-app/lamha/internal/i18n"
)

// Store is the persisted set of user keybinds.
type Store struct {
	mu     sync.RWMutex
	values map[ID]string
	path   string
}

var (
	currentMu sync.RWMutex
	current   *Store
)

// Load reads keybinds from disk, filling any missing entries with defaults.
func Load() *Store {
	store := &Store{values: defaults(), path: configPath()}
	if data, err := os.ReadFile(store.path); err == nil {
		var raw map[string]string
		if json.Unmarshal(data, &raw) == nil {
			for id, accel := range raw {
				if _, ok := Lookup(ID(id)); ok {
					store.values[ID(id)] = accel
				}
			}
		}
	}
	currentMu.Lock()
	current = store
	currentMu.Unlock()
	return store
}

// Current returns the process-wide keybind store, loading it if needed.
func Current() *Store {
	currentMu.RLock()
	store := current
	currentMu.RUnlock()
	if store != nil {
		return store
	}
	return Load()
}

// Accel returns the GTK accelerator string for id.
func (s *Store) Accel(id ID) string {
	if s == nil {
		if item, ok := Lookup(id); ok {
			return item.Default
		}
		return ""
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if accel, ok := s.values[id]; ok {
		return accel
	}
	if item, ok := Lookup(id); ok {
		return item.Default
	}
	return ""
}

// Set writes one shortcut. An empty accel unbinds the action.
func (s *Store) Set(id ID, accel string) error {
	if _, ok := Lookup(id); !ok {
		return fmt.Errorf(i18n.T("unknown shortcut %q"), id)
	}
	accel = strings.TrimSpace(accel)
	if accel != "" {
		if other, ok := s.usedBy(accel, id); ok {
			item, _ := Lookup(other)
			return fmt.Errorf("%s", i18n.Tf("already used by %s", i18n.T(item.Label)))
		}
	}
	s.mu.Lock()
	s.values[id] = accel
	s.mu.Unlock()
	return s.save()
}

// Reset restores one shortcut to its default.
func (s *Store) Reset(id ID) error {
	item, ok := Lookup(id)
	if !ok {
		return fmt.Errorf(i18n.T("unknown shortcut %q"), id)
	}
	return s.Set(id, item.Default)
}

// ResetAll restores every shortcut to its default.
func (s *Store) ResetAll() error {
	s.mu.Lock()
	s.values = defaults()
	s.mu.Unlock()
	return s.save()
}

// GNOMEAccels maps GNOME custom-keybinding IDs to GTK accelerators.
func (s *Store) GNOMEAccels() map[string]string {
	return map[string]string{
		"lamha-area":   s.Accel(CaptureArea),
		"lamha-window": "",
		"lamha-screen": s.Accel(CaptureScreen),
	}
}

func (s *Store) usedBy(accel string, skip ID) (ID, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	want := normalizeAccel(accel)
	for _, item := range Catalog() {
		if item.ID == skip {
			continue
		}
		got := s.values[item.ID]
		if got == "" {
			got = item.Default
		}
		if normalizeAccel(got) == want && want != "" {
			return item.ID, true
		}
	}
	return "", false
}

func (s *Store) save() error {
	s.mu.RLock()
	raw := make(map[string]string, len(s.values))
	for id, accel := range s.values {
		raw[string(id)] = accel
	}
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

func defaults() map[ID]string {
	out := make(map[ID]string, 24)
	for _, item := range Catalog() {
		out[item.ID] = item.Default
	}
	return out
}

func configPath() string {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "lamha-keybinds.json"
		}
		config = filepath.Join(home, ".config")
	}
	return filepath.Join(config, "lamha", "keybinds.json")
}

func normalizeAccel(accel string) string {
	s := strings.ToLower(strings.TrimSpace(accel))
	s = strings.ReplaceAll(s, "<primary>", "<control>")
	s = strings.ReplaceAll(s, "<mod1>", "<alt>")
	s = strings.ReplaceAll(s, "<meta>", "<super>")
	return s
}
