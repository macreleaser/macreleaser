package cli

import (
	"os"

	"github.com/macreleaser/macreleaser/pkg/config"
	"github.com/spf13/cobra"
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate example macreleaser configuration",
	Long: `Generate an example .macreleaser.yaml configuration file in the current directory.
This file contains all the basic configuration sections with example values.`,
	Run: runInit,
}

// runInit executes the init command
func runInit(cmd *cobra.Command, args []string) {
	logger := SetupLogger(GetDebugMode())
	configPath := ".macreleaser.yaml"

	// Check if config file already exists
	if _, err := os.Stat(configPath); err == nil {
		logger.Infof("Configuration file %s already exists", configPath)
		os.Exit(0)
	}

	// Create example configuration
	exampleConfig := config.ExampleConfig()

	// Save the configuration
	if err := config.SaveConfig(configPath, exampleConfig); err != nil {
		ExitWithErrorf(logger, "Failed to save configuration: %v", err)
	}

	logger.Infof("Example configuration created: %s", configPath)
	logger.Info("Edit this file to match your project requirements")
}
