package autostart

import (
	"fmt"
	"os"
	"path/filepath"
)

const desktopName = "io.github.lamha.Lamha.desktop"

// Path is the user autostart desktop file.
func Path() (string, error) {
	config := os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		config = filepath.Join(home, ".config")
	}
	return filepath.Join(config, "autostart", desktopName), nil
}

// Enabled reports whether Lamha is set to start in the background on login.
func Enabled() bool {
	path, err := Path()
	if err != nil {
		return false
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// SetEnabled writes or removes the autostart desktop file.
func SetEnabled(enabled bool) error {
	path, err := Path()
	if err != nil {
		return err
	}
	if !enabled {
		err := os.Remove(path)
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve lamha binary: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	contents := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=Lamha
Name[ar]=لمحة
Comment=Background screenshot capture
Comment[ar]=التقاط لقطات الشاشة في الخلفية
Exec=%q --background
Icon=io.github.lamha.Lamha
Terminal=false
X-GNOME-Autostart-enabled=true
`, exe)
	return os.WriteFile(path, []byte(contents), 0o644)
}
