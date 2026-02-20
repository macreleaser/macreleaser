package pipe

import (
	"github.com/macreleaser/macreleaser/internal/pipe/archive"
	"github.com/macreleaser/macreleaser/internal/pipe/build"
	"github.com/macreleaser/macreleaser/internal/pipe/changelog"
	"github.com/macreleaser/macreleaser/internal/pipe/homebrew"
	"github.com/macreleaser/macreleaser/internal/pipe/notarize"
	"github.com/macreleaser/macreleaser/internal/pipe/project"
	"github.com/macreleaser/macreleaser/internal/pipe/release"
	"github.com/macreleaser/macreleaser/internal/pipe/sign"
)

// ValidationPipes contains all validation pipes, run by check and as the
// first stage of build/release/snapshot.
var ValidationPipes = []Piper{
	project.CheckPipe{},  // Validate project config
	build.CheckPipe{},    // Validate build config
	sign.CheckPipe{},     // Validate signing config
	notarize.CheckPipe{}, // Validate notarization config
	archive.CheckPipe{},   // Validate archive config
	changelog.CheckPipe{}, // Validate changelog config
	release.CheckPipe{},   // Validate release config
	homebrew.CheckPipe{}, // Validate homebrew config
}

// ExecutionPipes contains all execution pipes, run after validation
// succeeds in build/release/snapshot commands.
var ExecutionPipes = []Piper{
	build.Pipe{},      // Build and archive with xcodebuild
	sign.Pipe{},       // Code sign with Hardened Runtime
	notarize.Pipe{},   // Submit, wait, staple .app
	archive.Pipe{},    // Package stapled .app into zip/dmg
	changelog.Pipe{},  // Generate changelog from git history
	release.Pipe{},    // Create GitHub release and upload assets
	homebrew.Pipe{},   // Generate cask and commit to tap
}
