package cli

import (
	"fmt"
	"os"

	"github.com/macreleaser/macreleaser/pkg/version"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "macreleaser",
	Short:   "macOS app release automation",
	Version: version.VersionInfo(),
	Long: `MacReleaser automates the build, sign, notarize, and release process 
for macOS applications. It provides a configuration-driven approach to 
managing the complete release lifecycle from Xcode build to Homebrew distribution.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := cmd.Help(); err != nil {
			fmt.Fprintf(os.Stderr, "Error displaying help: %v\n", err)
			os.Exit(1)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() error {
	registerCommands()
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	return rootCmd.Execute()
}

// registerCommands initializes flags and registers all subcommands
func registerCommands() {
	// Set up persistent flags
	rootCmd.PersistentFlags().String("config", ".macreleaser.yaml", "config file path")
	rootCmd.PersistentFlags().Bool("debug", false, "enable debug mode")

	// Add all subcommands
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(buildCmd)
	rootCmd.AddCommand(releaseCmd)
	rootCmd.AddCommand(snapshotCmd)

	// --clean is available on build, release, and snapshot
	buildCmd.Flags().Bool("clean", false, "remove dist/ before building")
	releaseCmd.Flags().Bool("clean", false, "remove dist/ before building")
	snapshotCmd.Flags().Bool("clean", false, "remove dist/ before building")

	// --skip-notarize is available on build and snapshot (not release)
	buildCmd.Flags().Bool("skip-notarize", false, "skip notarization (for quick local pipeline validation)")
	snapshotCmd.Flags().Bool("skip-notarize", false, "skip notarization (for quick local pipeline validation)")
}

// GetConfigPath returns the config file path from flags
func GetConfigPath() string {
	configPath, _ := rootCmd.PersistentFlags().GetString("config")
	return configPath
}

// GetDebugMode returns debug mode flag value
func GetDebugMode() bool {
	debug, _ := rootCmd.PersistentFlags().GetBool("debug")
	return debug
}
