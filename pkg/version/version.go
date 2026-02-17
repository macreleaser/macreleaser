package version

import (
	"fmt"
	"runtime"
)

// Name of the application
const Name = "macreleaser"

// version, commit, and date are populated at build time via ldflags:
//
//	-X github.com/macreleaser/macreleaser/pkg/version.version=...
//	-X github.com/macreleaser/macreleaser/pkg/version.commit=...
//	-X github.com/macreleaser/macreleaser/pkg/version.date=...
var (
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

// VersionInfo returns complete version information
func VersionInfo() string {
	return fmt.Sprintf("%s version %s\nCommit: %s\nBuilt: %s\nGo version: %s (%s/%s)",
		Name, version, commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
}

// ShortVersion returns name and version only
func ShortVersion() string {
	return fmt.Sprintf("%s %s", Name, version)
}
