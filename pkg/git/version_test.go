package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveVersion(t *testing.T) {
	dir := setupGitRepo(t, "v1.2.3")

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(original) }()

	version, err := ResolveVersion()
	if err != nil {
		t.Fatalf("ResolveVersion() error = %v", err)
	}
	if version != "v1.2.3" {
		t.Errorf("ResolveVersion() = %q, want %q", version, "v1.2.3")
	}
}

func TestResolveVersionNoTags(t *testing.T) {
	dir := setupGitRepo(t, "")

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(original) }()

	_, err = ResolveVersion()
	if err == nil {
		t.Fatal("ResolveVersion() expected error for repo with no tags")
	}
	if !strings.Contains(err.Error(), "no git tags found") {
		t.Errorf("ResolveVersion() error = %v, want error containing 'no git tags found'", err)
	}
}

func TestResolveVersionLatestTag(t *testing.T) {
	dir := setupGitRepo(t, "v1.0.0")

	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(original) }()

	// Add another commit and tag
	writeFile(t, filepath.Join(dir, "file2.txt"), "content2")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "second commit")
	runGit(t, dir, "tag", "v2.0.0")

	version, err := ResolveVersion()
	if err != nil {
		t.Fatalf("ResolveVersion() error = %v", err)
	}
	if version != "v2.0.0" {
		t.Errorf("ResolveVersion() = %q, want %q", version, "v2.0.0")
	}
}

// setupGitRepo creates a temporary git repo and returns its path.
// If git init is not possible (e.g., in a restricted sandbox), the test is skipped.
func setupGitRepo(t *testing.T, tag string) string {
	t.Helper()

	dir := t.TempDir()

	// Try to init - if sandbox prevents this, skip the test
	if err := tryRunGit(dir, "init", "--template="); err != nil {
		t.Skipf("Skipping: git init not available in this environment: %v", err)
	}

	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	// Create an initial commit (tags require at least one commit)
	writeFile(t, filepath.Join(dir, "file.txt"), "content")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")

	if tag != "" {
		runGit(t, dir, "tag", tag)
	}

	return dir
}

func tryRunGit(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	_, err := cmd.CombinedOutput()
	return err
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
