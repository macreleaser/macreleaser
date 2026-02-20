package build

import (
	"fmt"
	"os/exec"
	"strings"
)

// XcodebuildArgs holds the arguments needed to invoke xcodebuild archive.
type XcodebuildArgs struct {
	Scheme        string // -scheme
	Workspace     string // -workspace (for .xcworkspace) or -project (for .xcodeproj)
	WorkspaceType WorkspaceType
	Configuration string // -configuration
	ArchivePath   string // -archivePath
	Version       string // MARKETING_VERSION build setting (CFBundleShortVersionString)
	BuildNumber   string // CURRENT_PROJECT_VERSION build setting (CFBundleVersion)
}

// BuildArchiveArgs constructs the argument list for xcodebuild archive.
func BuildArchiveArgs(args XcodebuildArgs) []string {
	var cmdArgs []string

	if args.Workspace != "" {
		switch args.WorkspaceType {
		case Workspace:
			cmdArgs = append(cmdArgs, "-workspace", args.Workspace)
		case Project:
			cmdArgs = append(cmdArgs, "-project", args.Workspace)
		}
	}

	if args.Scheme != "" {
		cmdArgs = append(cmdArgs, "-scheme", args.Scheme)
	}

	if args.Configuration != "" {
		cmdArgs = append(cmdArgs, "-configuration", args.Configuration)
	}

	if args.ArchivePath != "" {
		cmdArgs = append(cmdArgs, "-archivePath", args.ArchivePath)
	}

	cmdArgs = append(cmdArgs, "archive")

	// Skip code signing during archive — macreleaser re-signs with codesign
	cmdArgs = append(cmdArgs, "CODE_SIGN_IDENTITY=-")

	// Build settings go after the action
	if args.Version != "" {
		cmdArgs = append(cmdArgs, "MARKETING_VERSION="+args.Version)
	}
	if args.BuildNumber != "" {
		cmdArgs = append(cmdArgs, "CURRENT_PROJECT_VERSION="+args.BuildNumber)
	}

	return cmdArgs
}

// RunXcodebuild executes xcodebuild with the given arguments.
// Returns combined stdout/stderr output and any error.
func RunXcodebuild(args XcodebuildArgs) (string, error) {
	// Check that xcodebuild is available
	if _, err := exec.LookPath("xcodebuild"); err != nil {
		return "", fmt.Errorf("xcodebuild not found — install Xcode Command Line Tools with: xcode-select --install")
	}

	cmdArgs := BuildArchiveArgs(args)
	cmd := exec.Command("xcodebuild", cmdArgs...)

	out, err := cmd.CombinedOutput()
	output := string(out)

	if err != nil {
		// Provide actionable error messages
		if strings.Contains(output, "xcodebuild: error: The workspace") {
			return output, fmt.Errorf("workspace not found — check project.workspace in your config: %w", err)
		}
		if strings.Contains(output, "xcodebuild: error: The project") {
			return output, fmt.Errorf("project not found — check project.workspace in your config: %w", err)
		}
		if strings.Contains(output, "Scheme") && strings.Contains(output, "is not currently configured") {
			return output, fmt.Errorf("scheme %q not found — check project.scheme in your config: %w", args.Scheme, err)
		}
		return output, fmt.Errorf("xcodebuild archive failed: %w", err)
	}

	return output, nil
}
