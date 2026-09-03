// Package version is Lamha's calendar version (YY.0M.MICRO).
package version

// Version is the release name shown on the command line and in Settings.
// CI and `make build` override it with -ldflags from the VERSION file.
// Keep the fallback synchronized with VERSION so direct source builds show the
// same release to users.
var Version = "26.09.0"

// String is the version shown on the command line and in Settings.
func String() string {
	return Version
}
