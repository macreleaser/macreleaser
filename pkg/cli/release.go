package cli

import (
	"github.com/spf13/cobra"
)

// releaseCmd represents the release command
var releaseCmd = &cobra.Command{
	Use:   "release",
	Short: "Full release process including Homebrew",
	Long: `Run the complete release process.
This will build, sign, notarize, package, and release your application
to GitHub. Requires GITHUB_TOKEN environment variable for authentication.`,
	Run: func(cmd *cobra.Command, args []string) {
		runPipelineCommand("Release", requireGitVersion)
	},
}
