// Package version is Lamha's calendar version (YY.0M.MICRO).
package version

// Version is the release name. CI and `make build` override it with -ldflags
// from the VERSION file. Untagged local builds stay 0.0.0-dev.
var Version = "0.0.0-dev"

// String is the version shown on the command line and in Settings.
func String() string {
	return Version
}
