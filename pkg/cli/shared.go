package cli

import (
	"context"
	"fmt"
	"os"

	"github.com/macreleaser/macreleaser/pkg/config"
	macContext "github.com/macreleaser/macreleaser/pkg/context"
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
		logger.SetFormatter(&logrus.TextFormatter{
			DisableTimestamp: true,
		})
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

// runPipelineCommand is the shared implementation for build, release, and snapshot.
// resolveVersion returns the version string to use; commandName appears in error messages.
func runPipelineCommand(commandName string, resolveVersion func(*logrus.Logger) string, opts ...pipelineOption) {
	logger := SetupLogger(GetDebugMode())
	configPath := GetConfigPath()

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		ExitWithErrorf(logger, "Failed to load configuration: %v", err)
	}

	logger.Info("Configuration loaded successfully")

	version := resolveVersion(logger)

	ctx := macContext.NewContext(context.Background(), cfg, logger)
	ctx.Version = version
	for _, opt := range opts {
		opt(ctx)
	}

	if err := pipeline.RunAll(ctx); err != nil {
		ExitWithErrorf(logger, "%s failed: %v", commandName, err)
	}

	printArtifactSummary(ctx)
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
