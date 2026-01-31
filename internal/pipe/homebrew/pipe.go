package homebrew

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/macreleaser/macreleaser/pkg/context"
	gh "github.com/macreleaser/macreleaser/pkg/github"
	"github.com/macreleaser/macreleaser/pkg/homebrew"
)

// skipError signals an intentional skip. It satisfies the pipe.IsSkip interface
// checked by the pipeline runner, without importing pkg/pipe (which would cause
// an import cycle through pkg/pipe/registry.go).
type skipError string

func (e skipError) Error() string { return string(e) }
func (e skipError) IsSkip() bool  { return true }

// Pipe generates a Homebrew cask file and optionally commits it to a custom tap.
type Pipe struct{}

func (Pipe) String() string { return "generating Homebrew cask" }

func (Pipe) Run(ctx *context.Context) error {
	if ctx.SkipPublish {
		return skipError("homebrew publishing skipped")
	}

	if len(ctx.Artifacts.Packages) == 0 {
		return fmt.Errorf("no packages found for Homebrew cask — ensure the archive step completed successfully")
	}

	if ctx.Artifacts.AppPath == "" {
		return fmt.Errorf("no .app path found — ensure the build step completed successfully")
	}

	// Select the best archive for the cask (prefer .zip)
	packagePath, err := homebrew.SelectPackage(ctx.Artifacts.Packages)
	if err != nil {
		return err
	}

	filename := filepath.Base(packagePath)
	ctx.Logger.Infof("Computing SHA256 hash of %s", filename)

	hash, err := homebrew.ComputeSHA256(packagePath)
	if err != nil {
		return fmt.Errorf("failed to compute SHA256 for %s: %w", filename, err)
	}

	owner := ctx.Config.Release.GitHub.Owner
	repo := ctx.Config.Release.GitHub.Repo
	assetURL := homebrew.BuildAssetURL(owner, repo, ctx.Version, filename)

	data := homebrew.CaskData{
		Token:    ctx.Config.Homebrew.Cask.Name,
		Version:  strings.TrimPrefix(ctx.Version, "v"),
		SHA256:   hash,
		URL:      assetURL,
		Name:     ctx.Config.Project.Name,
		Desc:     ctx.Config.Homebrew.Cask.Desc,
		Homepage: ctx.Config.Homebrew.Cask.Homepage,
		AppName:  filepath.Base(ctx.Artifacts.AppPath),
		License:  ctx.Config.Homebrew.Cask.License,
	}

	caskContent, err := homebrew.RenderCask(data)
	if err != nil {
		return err
	}

	// Write local cask file
	localPath := filepath.Join(ctx.Artifacts.BuildOutputDir, data.Token+".rb")
	if err := os.WriteFile(localPath, []byte(caskContent), 0644); err != nil {
		return fmt.Errorf("failed to write cask file: %w", err)
	}
	ctx.Artifacts.HomebrewCaskPath = localPath
	ctx.Logger.Infof("Generated cask file: %s", localPath)

	// Commit to custom tap if configured
	if isTapConfigured(ctx.Config.Homebrew.Tap) {
		if err := commitToTap(ctx, data, caskContent); err != nil {
			return err
		}
	}

	ctx.Logger.Infof("Homebrew cask generated: %s", data.Token)
	return nil
}

func commitToTap(ctx *context.Context, data homebrew.CaskData, caskContent string) error {
	tapOwner := ctx.Config.Homebrew.Tap.Owner
	tapName := ctx.Config.Homebrew.Tap.Name

	// Create GitHub client from tap token if not already injected (e.g., by tests)
	if ctx.HomebrewClient == nil {
		client, err := gh.NewClient(ctx.Config.Homebrew.Tap.Token)
		if err != nil {
			return fmt.Errorf("failed to create GitHub client for tap: %w", err)
		}
		ctx.HomebrewClient = client
	}

	caskPath := fmt.Sprintf("Casks/%s.rb", data.Token)
	content := []byte(caskContent)

	// Check if the file already exists (for update vs create)
	existing, err := ctx.HomebrewClient.GetFileContents(ctx.StdCtx, tapOwner, tapName, caskPath)
	if err == nil {
		// File exists — update it
		message := fmt.Sprintf("Update %s to %s", data.Token, data.Version)
		if err := ctx.HomebrewClient.UpdateFile(ctx.StdCtx, tapOwner, tapName, caskPath, message, content, existing.GetSHA()); err != nil {
			return fmt.Errorf("failed to commit cask to tap %s/%s: %w", tapOwner, tapName, err)
		}
		ctx.Logger.Infof("Updated cask in %s/%s: %s", tapOwner, tapName, caskPath)
	} else if strings.Contains(err.Error(), "404") {
		// File doesn't exist — create it
		message := fmt.Sprintf("Add %s %s", data.Token, data.Version)
		if err := ctx.HomebrewClient.CreateFile(ctx.StdCtx, tapOwner, tapName, caskPath, message, content); err != nil {
			return fmt.Errorf("failed to commit cask to tap %s/%s: %w", tapOwner, tapName, err)
		}
		ctx.Logger.Infof("Created cask in %s/%s: %s", tapOwner, tapName, caskPath)
	} else {
		return fmt.Errorf("failed to check existing cask in tap %s/%s: %w", tapOwner, tapName, err)
	}

	return nil
}
