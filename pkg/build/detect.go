package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WorkspaceType indicates whether the detected project is a workspace or project.
type WorkspaceType int

const (
	Workspace WorkspaceType = iota // .xcworkspace
	Project                        // .xcodeproj
)

// DetectedProject holds the result of workspace/project auto-detection.
type DetectedProject struct {
	Path string
	Type WorkspaceType
}

// DetectWorkspace auto-detects the Xcode workspace or project in the given directory.
// It looks for a single .xcworkspace first, then falls back to .xcodeproj.
// Returns an error if none found, or multiple are found.
func DetectWorkspace(dir string) (*DetectedProject, error) {
	// Look for .xcworkspace files first
	workspaces, err := findByExtension(dir, ".xcworkspace")
	if err != nil {
		return nil, fmt.Errorf("failed to scan for workspaces: %w", err)
	}

	// Filter out Pods workspace if there's a non-Pods workspace
	workspaces = filterPodsWorkspace(workspaces)

	if len(workspaces) == 1 {
		return &DetectedProject{
			Path: workspaces[0],
			Type: Workspace,
		}, nil
	}

	if len(workspaces) > 1 {
		return nil, fmt.Errorf(
			"multiple .xcworkspace files found: %s — set project.workspace in your config to specify which one to use",
			strings.Join(workspaces, ", "),
		)
	}

	// Fall back to .xcodeproj
	projects, err := findByExtension(dir, ".xcodeproj")
	if err != nil {
		return nil, fmt.Errorf("failed to scan for projects: %w", err)
	}

	if len(projects) == 1 {
		return &DetectedProject{
			Path: projects[0],
			Type: Project,
		}, nil
	}

	if len(projects) > 1 {
		return nil, fmt.Errorf(
			"multiple .xcodeproj files found: %s — set project.workspace in your config to specify which one to use",
			strings.Join(projects, ", "),
		)
	}

	return nil, fmt.Errorf("no .xcworkspace or .xcodeproj found in %s — ensure you are in the correct directory or set project.workspace in your config", dir)
}

// findByExtension returns all entries in dir that have the given extension.
// Only looks at the top level (no recursion).
func findByExtension(dir, ext string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var matches []string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ext {
			matches = append(matches, entry.Name())
		}
	}
	return matches, nil
}

// filterPodsWorkspace removes Pods.xcworkspace if there's another workspace present.
func filterPodsWorkspace(workspaces []string) []string {
	if len(workspaces) <= 1 {
		return workspaces
	}

	var filtered []string
	for _, ws := range workspaces {
		if ws != "Pods.xcworkspace" {
			filtered = append(filtered, ws)
		}
	}

	if len(filtered) == 0 {
		return workspaces
	}
	return filtered
}
