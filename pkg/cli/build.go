package cli

import (
	"github.com/macreleaser/macreleaser/pkg/git"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build and archive project",
	Long: `Build and archive the Xcode project.
This command validates configuration, builds with xcodebuild, extracts
the .app from the archive, and packages it into the configured formats.`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := []pipelineOption{withSkipPublish()}
		if clean, _ := cmd.Flags().GetBool("clean"); clean {
			opts = append(opts, withClean())
		}
		if skip, _ := cmd.Flags().GetBool("skip-notarize"); skip {
			opts = append(opts, withSkipNotarize())
		}
		runPipelineCommand("Build", requireGitVersion, opts...)
	},
}

// requireGitVersion resolves the version from git tags, exiting on failure.
func requireGitVersion(logger *logrus.Logger) string {
	version, err := git.ResolveVersion()
	if err != nil {
		ExitWithErrorf(logger, "Failed to resolve version: %v", err)
	}
	logger.Infof("Version: %s", version)
	return version
}
