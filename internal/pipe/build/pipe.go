package build

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/macreleaser/macreleaser/pkg/build"
	"github.com/macreleaser/macreleaser/pkg/context"
)

// Pipe executes the Xcode build, producing an .xcarchive and extracting the .app.
type Pipe struct{}

func (Pipe) String() string { return "building project" }

func (Pipe) Run(ctx *context.Context) error {
	cfg := ctx.Config

	// Validate path components to prevent traversal via config or git tags
	if !filepath.IsLocal(cfg.Project.Name) {
		return fmt.Errorf("project.name contains a path traversal or absolute path: %q", cfg.Project.Name)
	}
	if !filepath.IsLocal(ctx.Version) {
		return fmt.Errorf("version contains a path traversal or absolute path: %q", ctx.Version)
	}

	// Determine output directory
	outputDir := filepath.Join("dist", cfg.Project.Name, ctx.Version)
	ctx.Artifacts.BuildOutputDir = outputDir

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Detect workspace/project if not configured
	workspace, wsType, err := resolveWorkspace(ctx)
	if err != nil {
		return err
	}

	// Build archive path
	archivePath := filepath.Join(outputDir, cfg.Project.Scheme+".xcarchive")

	ctx.Logger.Infof("Building scheme %q with configuration %q", cfg.Project.Scheme, cfg.Build.Configuration)
	ctx.Logger.Infof("Archive path: %s", archivePath)

	// Run xcodebuild
	args := build.XcodebuildArgs{
		Scheme:        cfg.Project.Scheme,
		Workspace:     workspace,
		WorkspaceType: wsType,
		Configuration: cfg.Build.Configuration,
		ArchivePath:   archivePath,
	}

	output, err := build.RunXcodebuild(args)
	if err != nil {
		ctx.Logger.Debug(output)
		return fmt.Errorf("build failed: %w", err)
	}

	ctx.Logger.Debug(output)
	ctx.Artifacts.ArchivePath = archivePath

	// Extract .app from .xcarchive
	if err := extractApp(ctx, archivePath, outputDir); err != nil {
		return err
	}

	ctx.Logger.Infof("Build completed: %s", ctx.Artifacts.AppPath)
	return nil
}

// resolveWorkspace determines the workspace or project path to use.
func resolveWorkspace(ctx *context.Context) (string, build.WorkspaceType, error) {
	configured := ctx.Config.Project.Workspace

	if configured != "" {
		// Reject absolute paths and traversal sequences
		if !filepath.IsLocal(configured) {
			return "", 0, fmt.Errorf("project.workspace contains a path traversal or absolute path: %q", configured)
		}

		// User specified a workspace/project in config
		if strings.HasSuffix(configured, ".xcworkspace") {
			return configured, build.Workspace, nil
		}
		if strings.HasSuffix(configured, ".xcodeproj") {
			return configured, build.Project, nil
		}
		return "", 0, fmt.Errorf("project.workspace must end with .xcworkspace or .xcodeproj, got %q", configured)
	}

	// Auto-detect
	ctx.Logger.Info("Auto-detecting workspace/project...")
	cwd, err := os.Getwd()
	if err != nil {
		return "", 0, fmt.Errorf("failed to get working directory: %w", err)
	}

	detected, err := build.DetectWorkspace(cwd)
	if err != nil {
		return "", 0, err
	}

	ctx.Logger.Infof("Detected %s", detected.Path)
	return detected.Path, detected.Type, nil
}

// extractApp locates the .app inside the .xcarchive and copies it to the output directory.
func extractApp(ctx *context.Context, archivePath, outputDir string) error {
	appsDir := filepath.Join(archivePath, "Products", "Applications")

	entries, err := os.ReadDir(appsDir)
	if err != nil {
		return fmt.Errorf("failed to read .xcarchive Products/Applications: %w", err)
	}

	var appName string
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".app") {
			appName = entry.Name()
			break
		}
	}

	if appName == "" {
		return fmt.Errorf("no .app found in %s — the archive may have failed to produce an application", appsDir)
	}

	srcApp := filepath.Join(appsDir, appName)
	dstApp := filepath.Join(outputDir, appName)

	// Verify .app exists and is a directory
	info, err := os.Stat(srcApp)
	if err != nil {
		return fmt.Errorf("failed to stat .app at %s: %w", srcApp, err)
	}
	if !info.IsDir() {
		return fmt.Errorf(".app at %s is not a directory — the archive may be corrupted", srcApp)
	}

	// Copy using cp -R to preserve structure
	cmd := exec.Command("cp", "-R", srcApp, dstApp)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to copy .app to output directory: %s: %w", string(out), err)
	}

	ctx.Artifacts.AppPath = dstApp
	return nil
}
