// Package version is Lamha's calendar version (YY.0M.MICRO).
package version

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

// Version is the release name shown on the command line and in Settings.
// CI and `make build` override it with -ldflags from the VERSION file. When no
// linker value is supplied, initVersion reads VERSION for source builds.
var Version = "0.0.0-dev"

var releaseVersionPattern = regexp.MustCompile(`^[0-9]{2}\.[0-9]{2}\.[0-9]+$`)

func init() {
	initVersion()
}

func initVersion() {
	if Version != "0.0.0-dev" {
		return
	}

	for _, start := range versionSearchRoots() {
		for dir := start; dir != ""; dir = filepath.Dir(dir) {
			if raw, err := os.ReadFile(filepath.Join(dir, "VERSION")); err == nil {
				if value := strings.TrimSpace(string(raw)); releaseVersionPattern.MatchString(value) {
					Version = value
					return
				}
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
		}
	}
}

func versionSearchRoots() []string {
	roots := make([]string, 0, 3)
	if dir, err := os.Getwd(); err == nil {
		roots = append(roots, dir)
	}
	if executable, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(executable))
	}
	if _, file, _, ok := runtime.Caller(0); ok && filepath.IsAbs(file) {
		roots = append(roots, filepath.Dir(file))
	}
	return roots
}

// String is the version shown on the command line and in Settings.
func String() string {
	return Version
}
