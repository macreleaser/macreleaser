package changelog

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/macreleaser/macreleaser/pkg/changelog"
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/git"
)

// Pipe generates release notes from git history between tags.
type Pipe struct{}

func (Pipe) String() string { return "generating changelog" }

func (Pipe) Run(ctx *context.Context) error {
	if ctx.Config.Changelog.Disable {
		return skipError("changelog disabled")
	}

	gitRef := ctx.Git.Tag
	if gitRef == "" {
		gitRef = "HEAD"
	}

	prevTag, err := git.PreviousTag(gitRef)
	if err != nil {
		return fmt.Errorf("failed to find previous tag: %w", err)
	}

	commits, err := git.LogBetween(prevTag, gitRef)
	if err != nil {
		return fmt.Errorf("failed to get git log: %w", err)
	}

	content, err := changelog.Generate(ctx.Version, commits, ctx.Config.Changelog)
	if err != nil {
		return fmt.Errorf("failed to generate changelog: %w", err)
	}

	ctx.ReleaseNotes = content

	// Write CHANGELOG.md to dist/
	distDir := ctx.Artifacts.BuildOutputDir
	if distDir == "" {
		distDir = "dist"
	}
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return fmt.Errorf("failed to create dist directory: %w", err)
	}

	changelogPath := filepath.Join(distDir, "CHANGELOG.md")
	if err := os.WriteFile(changelogPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write changelog: %w", err)
	}

	ctx.Artifacts.ChangelogPath = changelogPath
	ctx.Logger.Infof("Changelog written to %s", changelogPath)

	return nil
}
