package notarize

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/macreleaser/macreleaser/pkg/archive"
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/notarize"
)

// skipError signals an intentional skip. It satisfies the pipe.IsSkip interface
// checked by the pipeline runner, without importing pkg/pipe (which would cause
// an import cycle through pkg/pipe/registry.go).
type skipError string

func (e skipError) Error() string { return string(e) }
func (e skipError) IsSkip() bool  { return true }

// Pipe executes Apple notarization on the signed .app bundle.
type Pipe struct{}

func (Pipe) String() string { return "notarizing application" }

func (Pipe) Run(ctx *context.Context) error {
	if ctx.SkipNotarize {
		return skipError("notarization skipped via --skip-notarize")
	}

	if ctx.Artifacts.AppPath == "" {
		return fmt.Errorf("no .app found to notarize — ensure the build and sign steps completed successfully")
	}

	appleID := ctx.Config.Notarize.AppleID
	teamID := ctx.Config.Notarize.TeamID
	password := ctx.Config.Notarize.Password

	// Create temporary ZIP for notarization submission
	appName := strings.TrimSuffix(filepath.Base(ctx.Artifacts.AppPath), ".app")
	zipPath := filepath.Join(ctx.Artifacts.BuildOutputDir, appName+"-notarize.zip")

	ctx.Logger.Info("Creating temporary ZIP for notarization submission")
	if err := archive.CreateZip(ctx.Artifacts.AppPath, zipPath); err != nil {
		return fmt.Errorf("failed to create temp ZIP for notarization: %w", err)
	}

	// Submit to Apple notary service
	ctx.Logger.Info("Submitting to Apple notary service (this may take several minutes)...")
	output, err := notarize.RunSubmit(zipPath, appleID, teamID, password)
	if err != nil {
		ctx.Logger.Debug(output)
		return fmt.Errorf("notarization failed: %w", err)
	}
	ctx.Logger.Debug(output)

	// Staple the notarization ticket to the .app
	ctx.Logger.Info("Stapling notarization ticket")
	output, err = notarize.RunStaple(ctx.Artifacts.AppPath)
	if err != nil {
		ctx.Logger.Debug(output)
		return fmt.Errorf("stapling failed: %w", err)
	}
	ctx.Logger.Debug(output)

	// Verify with Gatekeeper
	ctx.Logger.Info("Verifying Gatekeeper assessment")
	output, err = notarize.RunAssess(ctx.Artifacts.AppPath)
	if err != nil {
		ctx.Logger.Debug(output)
		return fmt.Errorf("Gatekeeper assessment failed: %w", err) //nolint:staticcheck // proper noun
	}
	ctx.Logger.Debug(output)

	// Clean up temp ZIP
	if removeErr := os.Remove(zipPath); removeErr != nil {
		ctx.Logger.Warnf("Failed to remove temp ZIP %s: %v", zipPath, removeErr)
	}

	ctx.Logger.Infof("Notarization complete: %s", ctx.Artifacts.AppPath)
	return nil
}
