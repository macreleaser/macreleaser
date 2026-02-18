package context

import (
	"context"

	"github.com/macreleaser/macreleaser/pkg/config"
	"github.com/macreleaser/macreleaser/pkg/git"
	"github.com/macreleaser/macreleaser/pkg/github"
	"github.com/sirupsen/logrus"
)

// Artifacts holds runtime output state populated by execution pipes.
// Subsequent pipes consume this data to chain build → archive → package steps.
type Artifacts struct {
	BuildOutputDir   string   // dist/
	ArchivePath      string   // path to .xcarchive
	AppPath          string   // path to extracted .app
	Packages         []string // paths to .zip, .dmg outputs
	ReleaseURL       string   // HTML URL of the created GitHub release
	HomebrewCaskPath string   // local path to the generated cask .rb file
}

// Context provides shared state for all pipes
type Context struct {
	StdCtx         context.Context        // Standard context for cancellation support
	Config         *config.Config
	Logger         *logrus.Logger
	Version        string                 // derived from git tag
	Git            git.GitInfo            // resolved git state
	Clean          bool                   // when true, remove dist/ before building
	Artifacts      *Artifacts             // populated by execution pipes
	SkipPublish    bool                   // when true, release pipe skips publishing
	SkipNotarize   bool                   // when true, notarize pipe skips notarization
	GitHubClient   github.ClientInterface // injectable GitHub API client
	HomebrewClient github.ClientInterface // injectable GitHub client for tap operations
}

// NewContext creates a new context with the given standard context, config, and logger.
// If stdCtx is nil, context.Background() is used.
func NewContext(stdCtx context.Context, cfg *config.Config, logger *logrus.Logger) *Context {
	if stdCtx == nil {
		stdCtx = context.Background()
	}
	return &Context{
		StdCtx:    stdCtx,
		Config:    cfg,
		Logger:    logger,
		Artifacts: &Artifacts{},
	}
}

// Done returns the done channel from the standard context for cancellation support
func (c *Context) Done() <-chan struct{} {
	return c.StdCtx.Done()
}

// Err returns the error from the standard context
func (c *Context) Err() error {
	return c.StdCtx.Err()
}
