package cli

import (
	"github.com/spf13/cobra"
)

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Full release process including Homebrew",
	Long: `Run the complete release process.
This will build, sign, notarize, and release your application to GitHub,
and update Homebrew casks. (Coming in later phases)`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := SetupLogger(GetDebugMode())
		logger.Info("Release command is not yet implemented")
		logger.Info("This will be available in later phases of macreleaser")
	},
}
