// Package prefs stores Lamha's user preferences.
package prefs

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
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

	ToolbarEdgeTop    = "top"
	ToolbarEdgeBottom = "bottom"
	ToolbarEdgeLeft   = "left"
	ToolbarEdgeRight  = "right"
	ToolbarEdgeFloat  = "float"

	ToolbarAlignStart  = "start"
	ToolbarAlignCenter = "center"
	ToolbarAlignEnd    = "end"

	DefaultToolbarEdge   = ToolbarEdgeTop
	DefaultToolbarOffset = 0.5
)

// Settings is the persisted preference file.
type Settings struct {
	MagnifierZoom    float64 `json:"magnifier_zoom"`
	Theme            string  `json:"theme"`
	Language         string  `json:"language"`
	ToolbarEdge      string  `json:"toolbar_edge"`
	ToolbarX         float64 `json:"toolbar_x"`
	ToolbarY         float64 `json:"toolbar_y"`
	ToolbarVertical  bool    `json:"toolbar_vertical"`
	ToolbarLocked    bool    `json:"toolbar_locked"`
	ToolbarSnapAlign bool    `json:"toolbar_snap_align"`
	RenumberSteps    bool    `json:"renumber_steps"`
	DeleteZeroSteps  bool    `json:"delete_zero_steps"`
	SaveDirectory    string  `json:"save_directory"`
}

type settingsFile struct {
	MagnifierZoom    float64  `json:"magnifier_zoom"`
	Theme            string   `json:"theme"`
	Language         string   `json:"language"`
	ToolbarEdge      string   `json:"toolbar_edge"`
	ToolbarOffset    *float64 `json:"toolbar_offset"`
	ToolbarX         *float64 `json:"toolbar_x"`
	ToolbarY         *float64 `json:"toolbar_y"`
	ToolbarVertical  *bool    `json:"toolbar_vertical"`
	ToolbarLocked    bool     `json:"toolbar_locked"`
	ToolbarSnapAlign *bool    `json:"toolbar_snap_align"`
	RenumberSteps    *bool    `json:"renumber_steps"`
	DeleteZeroSteps  *bool    `json:"delete_zero_steps"`
	SaveDirectory    string   `json:"save_directory"`
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
			var raw settingsFile
			if json.Unmarshal(data, &raw) == nil {
				live.current = settingsFromFile(raw)
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

// ToolbarEdge is top, bottom, left, right, or float.
func (s *Store) ToolbarEdge() string {
	if s == nil {
		return DefaultToolbarEdge
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return NormalizeToolbarEdge(s.current.ToolbarEdge)
}

// ToolbarX is 0..1. 0.5 is the toolbar's visual center on a horizontal edge.
func (s *Store) ToolbarX() float64 {
	if s == nil {
		return DefaultToolbarOffset
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ClampToolbarOffset(s.current.ToolbarX)
}

// ToolbarY is 0..1. 0.5 is the toolbar's visual center on a vertical edge.
func (s *Store) ToolbarY() float64 {
	if s == nil {
		return DefaultToolbarOffset
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return ClampToolbarOffset(s.current.ToolbarY)
}

// ToolbarOffset is 0..1 along the snapped edge.
func (s *Store) ToolbarOffset() float64 {
	if ToolbarVertical(s.ToolbarEdge()) {
		return s.ToolbarY()
	}
	return s.ToolbarX()
}

// ToolbarStacked is the last vertical orientation, used while floating.
func (s *Store) ToolbarStacked() bool {
	if s == nil {
		return ToolbarVertical(DefaultToolbarEdge)
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ToolbarFloating(s.current.ToolbarEdge) {
		return s.current.ToolbarVertical
	}
	return ToolbarVertical(s.current.ToolbarEdge)
}

// SetToolbarPlacement writes and persists the capture toolbar dock.
func (s *Store) SetToolbarPlacement(edge string, x, y float64) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.ToolbarEdge = NormalizeToolbarEdge(edge)
	s.current.ToolbarX = ClampToolbarOffset(x)
	s.current.ToolbarY = ClampToolbarOffset(y)
	switch s.current.ToolbarEdge {
	case ToolbarEdgeLeft, ToolbarEdgeRight:
		s.current.ToolbarVertical = true
	case ToolbarEdgeTop, ToolbarEdgeBottom:
		s.current.ToolbarVertical = false
	}
	s.mu.Unlock()
	return s.save()
}

// SetToolbarDock writes placement and the last stacked orientation.
func (s *Store) SetToolbarDock(edge string, x, y float64, vertical bool) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.ToolbarEdge = NormalizeToolbarEdge(edge)
	s.current.ToolbarX = ClampToolbarOffset(x)
	s.current.ToolbarY = ClampToolbarOffset(y)
	switch s.current.ToolbarEdge {
	case ToolbarEdgeLeft, ToolbarEdgeRight:
		s.current.ToolbarVertical = true
	case ToolbarEdgeTop, ToolbarEdgeBottom:
		s.current.ToolbarVertical = false
	default:
		s.current.ToolbarVertical = vertical
	}
	s.mu.Unlock()
	return s.save()
}

// ToolbarLocked reports whether the capture toolbar cannot be dragged.
func (s *Store) ToolbarLocked() bool {
	if s == nil {
		return false
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.ToolbarLocked
}

// SetToolbarLocked writes and persists whether the toolbar can be dragged.
func (s *Store) SetToolbarLocked(locked bool) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.ToolbarLocked = locked
	s.mu.Unlock()
	return s.save()
}

// ToolbarSnapAlign reports whether a drop snaps to the middle or corners.
func (s *Store) ToolbarSnapAlign() bool {
	if s == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.ToolbarSnapAlign
}

// SetToolbarSnapAlign writes and persists drop snapping along the edge.
func (s *Store) SetToolbarSnapAlign(snap bool) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.ToolbarSnapAlign = snap
	s.mu.Unlock()
	return s.save()
}

// RenumberSteps reports whether changing one step number shifts later ones.
func (s *Store) RenumberSteps() bool {
	if s == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.RenumberSteps
}

// SetRenumberSteps writes and persists whether later steps follow a change.
func (s *Store) SetRenumberSteps(renumber bool) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.RenumberSteps = renumber
	s.mu.Unlock()
	return s.save()
}

// DeleteZeroSteps reports whether turning a step number to 0 removes it.
func (s *Store) DeleteZeroSteps() bool {
	if s == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.current.DeleteZeroSteps
}

// SetDeleteZeroSteps writes and persists whether 0-numbered steps are removed.
func (s *Store) SetDeleteZeroSteps(deleteZero bool) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.DeleteZeroSteps = deleteZero
	s.mu.Unlock()
	return s.save()
}

// SaveDirectory is where new screenshots are written.
func (s *Store) SaveDirectory() string {
	if s == nil {
		return DefaultSaveDirectory()
	}
	s.mu.RLock()
	dir := s.current.SaveDirectory
	s.mu.RUnlock()
	return ResolveSaveDirectory(dir)
}

// SaveDirectoryIsDefault reports whether the save folder is still Pictures/Screenshots.
func (s *Store) SaveDirectoryIsDefault() bool {
	if s == nil {
		return true
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return strings.TrimSpace(s.current.SaveDirectory) == ""
}

// SetSaveDirectory writes and persists the screenshot folder. An empty path
// restores Pictures/Screenshots.
func (s *Store) SetSaveDirectory(dir string) error {
	if s == nil {
		return nil
	}
	dir = strings.TrimSpace(dir)
	if dir != "" {
		resolved := ResolveSaveDirectory(dir)
		if resolved == DefaultSaveDirectory() {
			dir = ""
		} else {
			dir = resolved
		}
	}
	s.mu.Lock()
	s.current.SaveDirectory = dir
	s.mu.Unlock()
	return s.save()
}

// ResetToolbar restores the capture toolbar to the top center.
func (s *Store) ResetToolbar() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.current.ToolbarEdge = DefaultToolbarEdge
	s.current.ToolbarX = DefaultToolbarOffset
	s.current.ToolbarY = DefaultToolbarOffset
	s.current.ToolbarVertical = false
	s.current.ToolbarLocked = false
	s.current.ToolbarSnapAlign = true
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
		MagnifierZoom:    DefaultMagnifierZoom,
		Theme:            ThemeSystem,
		Language:         LanguageSystem,
		ToolbarEdge:      DefaultToolbarEdge,
		ToolbarX:         DefaultToolbarOffset,
		ToolbarY:         DefaultToolbarOffset,
		ToolbarVertical:  false,
		ToolbarLocked:    false,
		ToolbarSnapAlign: true,
		RenumberSteps:    true,
		DeleteZeroSteps:  true,
	}
}

func settingsFromFile(raw settingsFile) Settings {
	out := defaults()
	out.MagnifierZoom = ClampZoom(raw.MagnifierZoom)
	out.Theme = NormalizeTheme(raw.Theme)
	out.Language = NormalizeLanguage(raw.Language)
	if raw.ToolbarEdge != "" {
		out.ToolbarEdge = NormalizeToolbarEdge(raw.ToolbarEdge)
	}
	if raw.ToolbarX != nil {
		out.ToolbarX = ClampToolbarOffset(*raw.ToolbarX)
	} else if raw.ToolbarOffset != nil && !ToolbarVertical(out.ToolbarEdge) {
		out.ToolbarX = ClampToolbarOffset(*raw.ToolbarOffset)
	}
	if raw.ToolbarY != nil {
		out.ToolbarY = ClampToolbarOffset(*raw.ToolbarY)
	} else if raw.ToolbarOffset != nil && ToolbarVertical(out.ToolbarEdge) {
		out.ToolbarY = ClampToolbarOffset(*raw.ToolbarOffset)
	}
	if raw.ToolbarVertical != nil {
		out.ToolbarVertical = *raw.ToolbarVertical
	} else {
		out.ToolbarVertical = ToolbarVertical(out.ToolbarEdge)
	}
	out.ToolbarLocked = raw.ToolbarLocked
	if raw.ToolbarSnapAlign != nil {
		out.ToolbarSnapAlign = *raw.ToolbarSnapAlign
	}
	if raw.RenumberSteps != nil {
		out.RenumberSteps = *raw.RenumberSteps
	}
	if raw.DeleteZeroSteps != nil {
		out.DeleteZeroSteps = *raw.DeleteZeroSteps
	}
	if raw.SaveDirectory != "" {
		out.SaveDirectory = ResolveSaveDirectory(raw.SaveDirectory)
		if out.SaveDirectory == DefaultSaveDirectory() {
			out.SaveDirectory = ""
		}
	}
	return out
}

// DefaultSaveDirectory is ~/Pictures/Screenshots, or the XDG Pictures folder.
func DefaultSaveDirectory() string {
	return filepath.Join(userPicturesDir(), "Screenshots")
}

// ResolveSaveDirectory expands and cleans a user-supplied folder. Empty or
// relative paths fall back to DefaultSaveDirectory.
func ResolveSaveDirectory(dir string) string {
	dir = expandHome(strings.TrimSpace(dir))
	if dir == "" || !filepath.IsAbs(dir) {
		return DefaultSaveDirectory()
	}
	return filepath.Clean(dir)
}

func userPicturesDir() string {
	if dir := expandHome(strings.TrimSpace(os.Getenv("XDG_PICTURES_DIR"))); dir != "" && filepath.IsAbs(dir) {
		return filepath.Clean(dir)
	}
	if dir := picturesFromUserDirs(); dir != "" {
		return dir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "Pictures"
	}
	return filepath.Join(home, "Pictures")
}

func picturesFromUserDirs() string {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		config = filepath.Join(home, ".config")
	}
	data, err := os.ReadFile(filepath.Join(config, "user-dirs.dirs"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != "XDG_PICTURES_DIR" {
			continue
		}
		val = strings.Trim(strings.TrimSpace(val), `"`)
		dir := expandHome(val)
		if dir != "" && filepath.IsAbs(dir) {
			return filepath.Clean(dir)
		}
	}
	return ""
}

func expandHome(path string) string {
	if path == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	path = strings.ReplaceAll(path, "$HOME", home)
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
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

// NormalizeToolbarEdge accepts top, bottom, left, or right.
func NormalizeToolbarEdge(edge string) string {
	switch strings.ToLower(strings.TrimSpace(edge)) {
	case ToolbarEdgeBottom, ToolbarEdgeLeft, ToolbarEdgeRight, ToolbarEdgeFloat:
		return strings.ToLower(strings.TrimSpace(edge))
	default:
		return ToolbarEdgeTop
	}
}

// ClampToolbarOffset keeps the along-edge position in 0..1.
func ClampToolbarOffset(offset float64) float64 {
	if math.IsNaN(offset) {
		return DefaultToolbarOffset
	}
	return math.Min(1, math.Max(0, offset))
}

// ToolbarVertical reports whether the toolbar should stack its tools.
func ToolbarVertical(edge string) bool {
	switch NormalizeToolbarEdge(edge) {
	case ToolbarEdgeLeft, ToolbarEdgeRight:
		return true
	default:
		return false
	}
}

// ToolbarFloating reports whether the toolbar is free of the screen edges.
func ToolbarFloating(edge string) bool {
	return NormalizeToolbarEdge(edge) == ToolbarEdgeFloat
}

// ToolbarOffsetForAlign maps start/center/end to a stored offset.
func ToolbarOffsetForAlign(align string) float64 {
	switch align {
	case ToolbarAlignStart:
		return 0
	case ToolbarAlignEnd:
		return 1
	default:
		return DefaultToolbarOffset
	}
}

// ToolbarAlignForOffset maps a stored offset to start, center, or end.
func ToolbarAlignForOffset(offset float64) string {
	offset = ClampToolbarOffset(offset)
	switch {
	case offset <= 1.0/3:
		return ToolbarAlignStart
	case offset >= 2.0/3:
		return ToolbarAlignEnd
	default:
		return ToolbarAlignCenter
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
