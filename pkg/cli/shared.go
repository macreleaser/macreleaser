package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/macreleaser/macreleaser/pkg/config"
	macContext "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/git"
	"github.com/macreleaser/macreleaser/pkg/logging"
	"github.com/macreleaser/macreleaser/pkg/pipeline"
	"github.com/sirupsen/logrus"
)

// SetupLogger creates and configures a logger based on debug mode
func SetupLogger(debug bool) *logrus.Logger {
	logger := logrus.New()

	if debug {
		logger.SetLevel(logrus.DebugLevel)
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
			ForceColors:   true,
		})
	} else {
		logger.SetLevel(logrus.InfoLevel)
		logger.SetFormatter(&logging.BulletFormatter{})
	}

	return logger
}

// ExitWithErrorf logs an error with the provided logger and exits with code 1
func ExitWithErrorf(logger *logrus.Logger, format string, args ...interface{}) {
	logger.Errorf(format, args...)
	os.Exit(1)
}

// ExitWithErrorNoLoggerf prints an error to stderr and exits with code 1
func ExitWithErrorNoLoggerf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}

// pipelineOption configures the pipeline context before execution.
type pipelineOption func(*macContext.Context)

// withSkipPublish returns an option that sets SkipPublish on the context,
// preventing the release pipe from creating a GitHub release.
func withSkipPublish() pipelineOption {
	return func(ctx *macContext.Context) {
		ctx.SkipPublish = true
	}
}

// withSkipNotarize returns an option that sets SkipNotarize on the context,
// skipping notarization and disabling hardened runtime during signing.
func withSkipNotarize() pipelineOption {
	return func(ctx *macContext.Context) {
		ctx.SkipNotarize = true
	}
}

// withClean returns an option that sets Clean on the context,
// causing dist/ to be removed before building.
func withClean() pipelineOption {
	return func(ctx *macContext.Context) {
		ctx.Clean = true
	}
}

// runPipelineCommand is the shared implementation for build, release, and snapshot.
// resolveVersion returns the version string to use; commandName appears in error messages.
func runPipelineCommand(commandName string, resolveVersion func(*logrus.Logger) string, opts ...pipelineOption) {
	logger := SetupLogger(GetDebugMode())
	configPath := GetConfigPath()

	logger.WithField("action", "loading configuration").Info()
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		ExitWithErrorf(logger, "Failed to load configuration: %v", err)
	}

	// Resolve git state
	logger.WithField("action", "getting and validating git state").Info()
	gitInfo, err := git.ResolveGitInfo()
	if err != nil {
		ExitWithErrorf(logger, "Failed to resolve git state: %v", err)
	}
	logger.WithFields(logrus.Fields{
		"action": "git state",
		"commit": gitInfo.ShortCommit,
		"branch": gitInfo.Branch,
		"tag":    gitInfo.Tag,
		"dirty":  gitInfo.Dirty,
	}).Info()

	version := resolveVersion(logger)

	ctx := macContext.NewContext(context.Background(), cfg, logger)
	ctx.Version = version
	ctx.Git = gitInfo
	for _, opt := range opts {
		opt(ctx)
	}

	// Clean dist/ if requested
	if ctx.Clean {
		logger.Info("Cleaning distribution directory")
		if err := os.RemoveAll("dist"); err != nil {
			ExitWithErrorf(logger, "Failed to clean dist/: %v", err)
		}
	}

	start := time.Now()
	if err := pipeline.RunAll(ctx); err != nil {
		ExitWithErrorf(logger, "%s failed: %v", commandName, err)
	}
	elapsed := time.Since(start)

	printArtifactSummary(ctx)
	logger.Infof("%s succeeded after %s", strings.ToLower(commandName), formatDuration(elapsed))
}

// formatDuration formats a duration in a human-friendly way:
// sub-second -> "523ms", seconds -> "5s", minutes+seconds -> "1m32s".
func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	d = d.Round(time.Second)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	if s == 0 {
		return fmt.Sprintf("%dm", m)
	}
	return fmt.Sprintf("%dm%ds", m, s)
}

// printArtifactSummary prints a concise summary of produced artifacts.
func printArtifactSummary(ctx *macContext.Context) {
	ctx.Logger.Info("---")
	ctx.Logger.Infof("Build complete for %s %s", ctx.Config.Project.Name, ctx.Version)

	if ctx.Artifacts.AppPath != "" {
		ctx.Logger.Infof("  App: %s", ctx.Artifacts.AppPath)
	}

	for _, pkg := range ctx.Artifacts.Packages {
		ctx.Logger.Infof("  Package: %s", pkg)
	}

	if ctx.Artifacts.ReleaseURL != "" {
		ctx.Logger.Infof("  Release: %s", ctx.Artifacts.ReleaseURL)
	}

	if ctx.Artifacts.HomebrewCaskPath != "" {
		ctx.Logger.Infof("  Cask: %s", ctx.Artifacts.HomebrewCaskPath)
	}

	fmt.Println()
	ctx.Logger.Infof("Artifacts in: %s", ctx.Artifacts.BuildOutputDir)
}
