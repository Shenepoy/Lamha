// Package capture persists screenshots returned by the desktop portal.
package capture

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// SavedCapture identifies a screenshot copied into Lamha's local library.
type SavedCapture struct {
	Path      string
	URI       string
	CreatedAt time.Time
}

// Store owns Lamha's durable capture directory.
type Store struct {
	directory string
}

// NewStore creates a store in directory. If directory is empty, the user's
// XDG data directory is used.
func NewStore(directory string) (*Store, error) {
	if directory == "" {
		dataDir, err := userDataDir()
		if err != nil {
			return nil, err
		}
		directory = filepath.Join(dataDir, "lamha", "captures")
	}
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return nil, fmt.Errorf("create capture directory: %w", err)
	}
	return &Store{directory: directory}, nil
}

func userDataDir() (string, error) {
	if dataDir := os.Getenv("XDG_DATA_HOME"); dataDir != "" {
		if !filepath.IsAbs(dataDir) {
			return "", errors.New("XDG_DATA_HOME must be an absolute path")
		}
		return dataDir, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory for capture store: %w", err)
	}
	return filepath.Join(home, ".local", "share"), nil
}

// Directory returns the location where Lamha stores successful captures.
func (s *Store) Directory() string {
	return s.directory
}

// SaveURI copies a screenshot returned as a file URI into the capture store.
// Copying matters because a portal may grant access through a short-lived
// document path rather than an ordinary, permanent filesystem location.
func (s *Store) SaveURI(rawURI string) (SavedCapture, error) {
	sourcePath, err := pathFromFileURI(rawURI)
	if err != nil {
		return SavedCapture{}, err
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return SavedCapture{}, fmt.Errorf("open portal screenshot: %w", err)
	}
	defer source.Close()

	extension := imageExtension(sourcePath)
	createdAt := time.Now()
	name := fmt.Sprintf("Lamha-%s-%s%s", createdAt.Format("20060102-150405"), randomSuffix(), extension)
	destination := filepath.Join(s.directory, name)
	temporary, err := os.CreateTemp(s.directory, ".capture-*.tmp")
	if err != nil {
		return SavedCapture{}, fmt.Errorf("create temporary capture: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if _, err := io.Copy(temporary, source); err != nil {
		temporary.Close()
		return SavedCapture{}, fmt.Errorf("copy portal screenshot: %w", err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return SavedCapture{}, fmt.Errorf("set capture permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return SavedCapture{}, fmt.Errorf("finish capture file: %w", err)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return SavedCapture{}, fmt.Errorf("save capture: %w", err)
	}

	return SavedCapture{Path: destination, URI: fileURI(destination), CreatedAt: createdAt}, nil
}

// CopyURIToTemp copies a short-lived portal screenshot into a temporary file
// so Lamha's capture overlay can work after the portal path is revoked.
func CopyURIToTemp(rawURI string) (string, error) {
	sourcePath, err := pathFromFileURI(rawURI)
	if err != nil {
		return "", err
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		return "", fmt.Errorf("open portal screenshot: %w", err)
	}
	defer source.Close()

	temporary, err := os.CreateTemp("", "lamha-capture-*.png")
	if err != nil {
		return "", fmt.Errorf("create staging capture: %w", err)
	}
	if _, err := io.Copy(temporary, source); err != nil {
		temporary.Close()
		os.Remove(temporary.Name())
		return "", fmt.Errorf("copy portal screenshot: %w", err)
	}
	if err := temporary.Close(); err != nil {
		os.Remove(temporary.Name())
		return "", fmt.Errorf("finish staging capture: %w", err)
	}
	return temporary.Name(), nil
}

// SaveImage writes img as a PNG in the capture library.
func (s *Store) SaveImage(img image.Image) (SavedCapture, error) {
	createdAt := time.Now()
	name := fmt.Sprintf("Lamha-%s-%s.png", createdAt.Format("20060102-150405"), randomSuffix())
	destination := filepath.Join(s.directory, name)
	if err := WritePNG(destination, img); err != nil {
		return SavedCapture{}, err
	}
	return SavedCapture{Path: destination, URI: fileURI(destination), CreatedAt: createdAt}, nil
}

// WritePNG replaces path with a PNG encoding of img using the same atomic
// write used for portal captures.
func WritePNG(path string, img image.Image) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create capture directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".annotate-*.tmp")
	if err != nil {
		return fmt.Errorf("create temporary capture: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)

	if err := png.Encode(temporary, img); err != nil {
		temporary.Close()
		return fmt.Errorf("encode annotated capture: %w", err)
	}
	if err := temporary.Chmod(0o644); err != nil {
		temporary.Close()
		return fmt.Errorf("set capture permissions: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("finish capture file: %w", err)
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return fmt.Errorf("save annotated capture: %w", err)
	}
	return nil
}

// List returns saved captures, newest first. Temporary files and non-image
// names are ignored so a half-written save cannot appear in history.
func (s *Store) List() ([]SavedCapture, error) {
	entries, err := os.ReadDir(s.directory)
	if err != nil {
		return nil, fmt.Errorf("list captures: %w", err)
	}

	captures := make([]SavedCapture, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		if !hasKnownImageExt(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(s.directory, entry.Name())
		captures = append(captures, SavedCapture{
			Path:      path,
			URI:       fileURI(path),
			CreatedAt: info.ModTime(),
		})
	}

	sort.Slice(captures, func(i, j int) bool {
		if captures[i].CreatedAt.Equal(captures[j].CreatedAt) {
			return captures[i].Path > captures[j].Path
		}
		return captures[i].CreatedAt.After(captures[j].CreatedAt)
	})
	return captures, nil
}

func hasKnownImageExt(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png", ".jpg", ".jpeg", ".webp", ".tif", ".tiff":
		return true
	default:
		return false
	}
}

func pathFromFileURI(rawURI string) (string, error) {
	parsed, err := url.Parse(rawURI)
	if err != nil {
		return "", fmt.Errorf("parse portal screenshot URI: %w", err)
	}
	if parsed.Scheme != "file" {
		return "", fmt.Errorf("portal returned unsupported screenshot URI scheme %q", parsed.Scheme)
	}
	if parsed.Host != "" && parsed.Host != "localhost" {
		return "", fmt.Errorf("portal returned non-local screenshot URI host %q", parsed.Host)
	}
	if parsed.Path == "" {
		return "", errors.New("portal returned an empty screenshot path")
	}
	return filepath.FromSlash(parsed.Path), nil
}

func fileURI(path string) string {
	return (&url.URL{Scheme: "file", Path: path}).String()
}

func imageExtension(path string) string {
	switch extension := strings.ToLower(filepath.Ext(path)); extension {
	case ".png", ".jpg", ".jpeg", ".webp", ".tif", ".tiff":
		return extension
	default:
		return ".png"
	}
}

func randomSuffix() string {
	bytes := make([]byte, 4)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}
