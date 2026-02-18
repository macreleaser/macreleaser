package cli

import (
	"fmt"

	"github.com/macreleaser/macreleaser/pkg/git"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// snapshotCmd represents the snapshot command
var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Test release with snapshot version",
	Long: `Create a test release with snapshot versioning.
This allows you to test the release process without affecting
versioned releases. If no git tags exist, a snapshot version is generated.`,
	Run: func(cmd *cobra.Command, args []string) {
		opts := []pipelineOption{withSkipPublish()}
		if skip, _ := cmd.Flags().GetBool("skip-notarize"); skip {
			opts = append(opts, withSkipNotarize())
		}
		runPipelineCommand("Snapshot", snapshotVersion, opts...)
	},
}

// snapshotVersion resolves a snapshot version in goreleaser-style format:
// <tag>-SNAPSHOT-<shortcommit> or 0.0.0-SNAPSHOT-<shortcommit> when no tags exist.
func snapshotVersion(logger *logrus.Logger) string {
	short, err := git.ShortCommit()
	if err != nil {
		ExitWithErrorf(logger, "Failed to resolve git commit: %v", err)
	}

	tag, tagErr := git.ResolveVersion()
	if tagErr != nil {
		tag = "0.0.0"
	}

	version := fmt.Sprintf("%s-SNAPSHOT-%s", tag, short)
	logger.Infof("Version: %s (snapshot)", version)
	return version
}
