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

// All contains all registered pipes in execution order
var All = []Piper{
	project.Pipe{},  // Validate project config
	build.Pipe{},    // Validate build config
	sign.Pipe{},     // Validate signing config
	notarize.Pipe{}, // Validate notarization config
	archive.Pipe{},  // Validate archive config
	release.Pipe{},  // Validate release config
	homebrew.Pipe{}, // Validate homebrew config
}
