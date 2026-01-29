package cli

import (
	"github.com/spf13/cobra"
)

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Test release with snapshot version",
	Long: `Create a test release with snapshot versioning.
This allows you to test the release process without affecting
versioned releases. (Coming in later phases)`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := SetupLogger(GetDebugMode())
		logger.Info("Snapshot command is not yet implemented")
		logger.Info("This will be available in later phases of macreleaser")
	},
}
