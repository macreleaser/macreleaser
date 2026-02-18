package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// GitInfo holds the resolved git state for the current repository.
type GitInfo struct {
	Commit      string // full SHA
	ShortCommit string // abbreviated SHA
	Branch      string // current branch name
	Tag         string // latest tag (empty if none)
	Dirty       bool   // true if working tree has uncommitted changes
	CommitCount int    // total number of commits reachable from HEAD
}

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

// FullCommit returns the full SHA of HEAD.
func FullCommit() (string, error) {
	return gitOutput("rev-parse", "HEAD")
}

// ShortCommit returns the abbreviated SHA of HEAD.
func ShortCommit() (string, error) {
	return gitOutput("rev-parse", "--short", "HEAD")
}

// Branch returns the current branch name, or empty string if detached.
func Branch() (string, error) {
	out, err := gitOutput("rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return "", err
	}
	if out == "HEAD" {
		return "", nil // detached HEAD
	}
	return out, nil
}

// IsDirty returns true if the working tree has uncommitted changes.
func IsDirty() (bool, error) {
	out, err := gitOutput("status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// CommitCount returns the number of commits reachable from HEAD.
func CommitCount() (int, error) {
	out, err := gitOutput("rev-list", "--count", "HEAD")
	if err != nil {
		return 0, err
	}
	var count int
	if _, err := fmt.Sscanf(out, "%d", &count); err != nil {
		return 0, fmt.Errorf("failed to parse commit count %q: %w", out, err)
	}
	return count, nil
}

// ResolveGitInfo gathers the full git state for the current repository.
func ResolveGitInfo() (GitInfo, error) {
	info := GitInfo{}

	commit, err := FullCommit()
	if err != nil {
		return info, fmt.Errorf("failed to resolve git commit: %w", err)
	}
	info.Commit = commit

	short, err := ShortCommit()
	if err != nil {
		return info, fmt.Errorf("failed to resolve short commit: %w", err)
	}
	info.ShortCommit = short

	branch, err := Branch()
	if err != nil {
		return info, fmt.Errorf("failed to resolve git branch: %w", err)
	}
	info.Branch = branch

	dirty, err := IsDirty()
	if err != nil {
		return info, fmt.Errorf("failed to check dirty state: %w", err)
	}
	info.Dirty = dirty

	tag, _ := ResolveVersion() // ignore error — no tag is fine
	info.Tag = tag

	count, err := CommitCount()
	if err != nil {
		return info, fmt.Errorf("failed to resolve commit count: %w", err)
	}
	info.CommitCount = count

	return info, nil
}

// gitOutput runs a git command and returns its trimmed stdout.
func gitOutput(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return strings.TrimSpace(string(out)), nil
}
