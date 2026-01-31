package cli

import (
	"fmt"
	"time"

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
		runPipelineCommand("Snapshot", snapshotVersion, withSkipPublish())
	},
}

// snapshotVersion resolves a snapshot version, falling back to a timestamp if no tags exist.
func snapshotVersion(logger *logrus.Logger) string {
	version, err := git.ResolveVersion()
	if err != nil {
		version = fmt.Sprintf("snapshot-%s", time.Now().Format("20060102150405"))
		logger.Infof("No git tags found, using snapshot version: %s", version)
	} else {
		version = version + "-snapshot"
		logger.Infof("Version: %s", version)
	}
	return version
}
