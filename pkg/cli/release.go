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
and update Homebrew casks. Signing, notarization, and upload are not yet
enabled — currently this runs the same pipeline as build.`,
	Run: func(cmd *cobra.Command, args []string) {
		runPipelineCommand("Release", requireGitVersion)
	},
}
