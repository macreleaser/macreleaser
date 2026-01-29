package cli

import (
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build and archive project",
	Long: `Build and archive the Xcode project.
This command will build your project using xcodebuild and create archives 
for the specified architectures. (Coming in Phase 2)`,
	Run: func(cmd *cobra.Command, args []string) {
		logger := SetupLogger(GetDebugMode())
		logger.Info("Build command is not yet implemented")
		logger.Info("This will be available in Phase 2 of macreleaser")
	},
}
