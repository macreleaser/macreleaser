package cli

import (
	"context"

	"github.com/macreleaser/macreleaser/pkg/config"
	macContext "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/pipeline"
	"github.com/spf13/cobra"
)

// checkCmd represents the check command
var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Validate configuration file",
	Long: `Validate the .macreleaser.yaml configuration file.
This command checks for syntax errors, required fields, and validates
the configuration against expected patterns and constraints.`,
	Run: runCheck,
}

// runCheck executes the check command
func runCheck(cmd *cobra.Command, args []string) {
	logger := SetupLogger(GetDebugMode())
	configPath := GetConfigPath()

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		ExitWithErrorf(logger, "Failed to load configuration: %v", err)
	}

	logger.Info("Configuration loaded successfully")

	// Create context
	ctx := macContext.NewContext(context.Background(), cfg, logger)

	// Run validation pipeline only
	if err := pipeline.RunValidation(ctx); err != nil {
		ExitWithErrorf(logger, "Configuration validation failed: %v", err)
	}

	logger.Info("Configuration is valid")
}
