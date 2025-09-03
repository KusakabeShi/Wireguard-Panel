package version

import (
	"fmt"
	"runtime"
)

var (
	// Version is the current version of the application
	Version = "dev"
	// GitCommit is the git commit hash
	GitCommit = "unknown"
	// BuildDate is the build date
	BuildDate = "unknown"
	// BuildUser is the user who built the binary
	BuildUser = "unknown"
)

// GetVersionInfo returns version information
func GetVersionInfo() string {
	return fmt.Sprintf("WG-Panel %s (commit: %s, built: %s by %s, go: %s)",
		Version, GitCommit, BuildDate, BuildUser, runtime.Version())
}

// GetShortVersion returns just the version string
func GetShortVersion() string {
	return Version
}