package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// ResolveVersion derives the project version from the latest git tag
// using `git describe --tags`. Returns a clean version string.
// Returns an actionable error if no git tags exist.
func ResolveVersion() (string, error) {
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	out, err := cmd.Output()
	if err != nil {
		// Check if it's because no tags exist
		if exitErr, ok := err.(*exec.ExitError); ok {
			stderr := string(exitErr.Stderr)
			if strings.Contains(stderr, "No names found") || strings.Contains(stderr, "No tags") || strings.Contains(stderr, "fatal") {
				return "", fmt.Errorf("no git tags found — tag your release with `git tag v1.0.0`")
			}
		}
		// Check if git is not installed
		if _, pathErr := exec.LookPath("git"); pathErr != nil {
			return "", fmt.Errorf("git is not installed or not in PATH")
		}
		return "", fmt.Errorf("failed to resolve version from git tags: %w", err)
	}

	version := strings.TrimSpace(string(out))
	if version == "" {
		return "", fmt.Errorf("no git tags found — tag your release with `git tag v1.0.0`")
	}

	return version, nil
}
