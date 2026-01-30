package archive

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/macreleaser/macreleaser/pkg/archive"
	"github.com/macreleaser/macreleaser/pkg/context"
)

// Pipe packages the built .app into the configured archive formats (zip, dmg).
type Pipe struct{}

func (Pipe) String() string { return "packaging archives" }

func (Pipe) Run(ctx *context.Context) error {
	if ctx.Artifacts.AppPath == "" {
		return fmt.Errorf("no .app found to package — ensure the build step completed successfully")
	}

	cfg := ctx.Config
	outputDir := ctx.Artifacts.BuildOutputDir

	// Derive app name without extension for package naming
	appBase := filepath.Base(ctx.Artifacts.AppPath)
	appName := strings.TrimSuffix(appBase, ".app")

	for _, format := range cfg.Archive.Formats {
		switch format {
		case "zip":
			outputPath := filepath.Join(outputDir, fmt.Sprintf("%s-%s.zip", appName, ctx.Version))
			ctx.Logger.Infof("Creating ZIP: %s", outputPath)

			if err := archive.CreateZip(ctx.Artifacts.AppPath, outputPath); err != nil {
				return fmt.Errorf("ZIP packaging failed: %w", err)
			}

			ctx.Artifacts.Packages = append(ctx.Artifacts.Packages, outputPath)
			ctx.Logger.Infof("ZIP created: %s", outputPath)

		case "dmg":
			outputPath := filepath.Join(outputDir, fmt.Sprintf("%s-%s.dmg", appName, ctx.Version))
			volumeName := fmt.Sprintf("%s %s", appName, ctx.Version)
			ctx.Logger.Infof("Creating DMG: %s", outputPath)

			if err := archive.CreateDMG(ctx.Artifacts.AppPath, outputPath, volumeName); err != nil {
				return fmt.Errorf("DMG packaging failed: %w", err)
			}

			ctx.Artifacts.Packages = append(ctx.Artifacts.Packages, outputPath)
			ctx.Logger.Infof("DMG created: %s", outputPath)

		case "app":
			// The .app is already in the output directory
			ctx.Artifacts.Packages = append(ctx.Artifacts.Packages, ctx.Artifacts.AppPath)
			ctx.Logger.Infof("App bundle: %s", ctx.Artifacts.AppPath)
		}
	}

	return nil
}
