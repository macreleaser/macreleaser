package release

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	gogithub "github.com/google/go-github/github"
	"github.com/macreleaser/macreleaser/pkg/context"
	gh "github.com/macreleaser/macreleaser/pkg/github"
)

// skipError signals an intentional skip. It satisfies the pipe.IsSkip interface
// checked by the pipeline runner, without importing pkg/pipe (which would cause
// an import cycle through pkg/pipe/registry.go).
type skipError string

func (e skipError) Error() string { return string(e) }
func (e skipError) IsSkip() bool  { return true }

// Pipe creates a GitHub release and uploads archive packages as release assets.
type Pipe struct{}

func (Pipe) String() string { return "publishing GitHub release" }

func (Pipe) Run(ctx *context.Context) error {
	if ctx.SkipPublish {
		return skipError("publishing skipped")
	}

	if len(ctx.Artifacts.Packages) == 0 {
		return fmt.Errorf("no packages to release — ensure the archive step completed successfully")
	}

	// Create GitHub client if not already injected (e.g., by tests)
	if ctx.GitHubClient == nil {
		token := gh.GetGitHubToken()
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable is required for publishing — create a token at https://github.com/settings/tokens with 'repo' scope")
		}
		client, err := gh.NewClient(token)
		if err != nil {
			return fmt.Errorf("failed to create GitHub client: %w", err)
		}
		ctx.GitHubClient = client
	}

	owner := ctx.Config.Release.GitHub.Owner
	repo := ctx.Config.Release.GitHub.Repo
	releaseName := fmt.Sprintf("%s %s", ctx.Config.Project.Name, ctx.Version)

	release, err := ctx.GitHubClient.CreateRelease(ctx.StdCtx, owner, repo, &gogithub.RepositoryRelease{
		TagName: &ctx.Version,
		Name:    &releaseName,
		Draft:   &ctx.Config.Release.GitHub.Draft,
	})
	if err != nil {
		if strings.Contains(err.Error(), "already_exists") {
			return fmt.Errorf("release for tag %s already exists — delete the existing release or use a different version tag", ctx.Version)
		}
		return fmt.Errorf("failed to create GitHub release: %w", err)
	}

	ctx.Artifacts.ReleaseURL = release.GetHTMLURL()
	ctx.Logger.Infof("Created GitHub release: %s", releaseName)

	// Upload packages as release assets
	for _, pkg := range ctx.Artifacts.Packages {
		info, err := os.Stat(pkg)
		if err != nil || !info.Mode().IsRegular() {
			ctx.Logger.Warnf("Skipping %s: not a regular file (only files can be uploaded as release assets)", pkg)
			continue
		}

		contentType := gh.ContentTypeForAsset(pkg)
		if _, err := ctx.GitHubClient.UploadReleaseAsset(ctx.StdCtx, owner, repo, release.GetID(), pkg, contentType); err != nil {
			return fmt.Errorf("failed to upload asset %s: %w", filepath.Base(pkg), err)
		}
		ctx.Logger.Infof("Uploaded: %s", filepath.Base(pkg))
	}

	ctx.Logger.Infof("Release published: %s", ctx.Artifacts.ReleaseURL)
	return nil
}
