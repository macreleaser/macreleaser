package version

import (
	"fmt"
	"runtime"
)

const (
	// Name of the application
	Name = "macreleaser"
	// Version of the application (populated at build time)
	Version = "dev"
	// Commit hash (populated at build time)
	Commit = "unknown"
	// Build date (populated at build time)
	Date = "unknown"
)

// VersionInfo returns complete version information
func VersionInfo() string {
	return fmt.Sprintf("%s version %s\nCommit: %s\nBuilt: %s\nGo version: %s (%s/%s)",
		Name, Version, Commit, Date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// ShortVersion returns name and version only
func ShortVersion() string {
	return fmt.Sprintf("%s %s", Name, Version)
}
