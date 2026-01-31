package pipe

import (
	"github.com/macreleaser/macreleaser/internal/pipe/archive"
	"github.com/macreleaser/macreleaser/internal/pipe/build"
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
	archive.CheckPipe{},  // Validate archive config
	release.CheckPipe{},  // Validate release config
	homebrew.CheckPipe{}, // Validate homebrew config
}

// ExecutionPipes contains all execution pipes, run after validation
// succeeds in build/release/snapshot commands.
var ExecutionPipes = []Piper{
	build.Pipe{},   // Build and archive with xcodebuild
	sign.Pipe{},    // Code sign the .app bundle
	archive.Pipe{}, // Package into zip/dmg
}
