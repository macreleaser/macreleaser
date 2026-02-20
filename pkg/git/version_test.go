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

func TestFullCommit(t *testing.T) {
	dir := setupGitRepo(t, "v1.0.0")
	chdir(t, dir)

	commit, err := FullCommit()
	if err != nil {
		t.Fatalf("FullCommit() error = %v", err)
	}
	if len(commit) != 40 {
		t.Errorf("FullCommit() = %q, want 40-char SHA", commit)
	}
}

func TestShortCommit(t *testing.T) {
	dir := setupGitRepo(t, "v1.0.0")
	chdir(t, dir)

	short, err := ShortCommit()
	if err != nil {
		t.Fatalf("ShortCommit() error = %v", err)
	}
	if len(short) < 7 {
		t.Errorf("ShortCommit() = %q, want at least 7 chars", short)
	}
}

func TestBranch(t *testing.T) {
	dir := setupGitRepo(t, "")
	chdir(t, dir)

	branch, err := Branch()
	if err != nil {
		t.Fatalf("Branch() error = %v", err)
	}
	// Default branch is "main" or "master" depending on git config
	if branch == "" {
		t.Error("Branch() returned empty string, expected branch name")
	}
}

func TestIsDirty(t *testing.T) {
	dir := setupGitRepo(t, "")
	chdir(t, dir)

	// Clean repo
	dirty, err := IsDirty()
	if err != nil {
		t.Fatalf("IsDirty() error = %v", err)
	}
	if dirty {
		t.Error("IsDirty() = true, want false for clean repo")
	}

	// Make it dirty
	writeFile(t, filepath.Join(dir, "dirty.txt"), "dirty")

	dirty, err = IsDirty()
	if err != nil {
		t.Fatalf("IsDirty() error = %v", err)
	}
	if !dirty {
		t.Error("IsDirty() = false, want true for dirty repo")
	}
}

func TestCommitCount(t *testing.T) {
	dir := setupGitRepo(t, "")
	chdir(t, dir)

	count, err := CommitCount()
	if err != nil {
		t.Fatalf("CommitCount() error = %v", err)
	}
	if count != 1 {
		t.Errorf("CommitCount() = %d, want 1", count)
	}

	// Add another commit
	writeFile(t, filepath.Join(dir, "file2.txt"), "content2")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "second commit")

	count, err = CommitCount()
	if err != nil {
		t.Fatalf("CommitCount() error = %v", err)
	}
	if count != 2 {
		t.Errorf("CommitCount() = %d, want 2", count)
	}
}

func TestResolveGitInfo(t *testing.T) {
	dir := setupGitRepo(t, "v1.0.0")
	chdir(t, dir)

	info, err := ResolveGitInfo()
	if err != nil {
		t.Fatalf("ResolveGitInfo() error = %v", err)
	}
	if len(info.Commit) != 40 {
		t.Errorf("Commit = %q, want 40-char SHA", info.Commit)
	}
	if len(info.ShortCommit) < 7 {
		t.Errorf("ShortCommit = %q, want at least 7 chars", info.ShortCommit)
	}
	if info.Branch == "" {
		t.Error("Branch is empty")
	}
	if info.Tag != "v1.0.0" {
		t.Errorf("Tag = %q, want %q", info.Tag, "v1.0.0")
	}
	if info.Dirty {
		t.Error("Dirty = true, want false")
	}
	if info.CommitCount != 1 {
		t.Errorf("CommitCount = %d, want 1", info.CommitCount)
	}
}

func TestResolveGitInfoNoTags(t *testing.T) {
	dir := setupGitRepo(t, "")
	chdir(t, dir)

	info, err := ResolveGitInfo()
	if err != nil {
		t.Fatalf("ResolveGitInfo() error = %v", err)
	}
	if info.Tag != "" {
		t.Errorf("Tag = %q, want empty", info.Tag)
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

func TestPreviousTag(t *testing.T) {
	dir := setupGitRepo(t, "v1.0.0")
	chdir(t, dir)

	// Add another commit and tag
	writeFile(t, filepath.Join(dir, "file2.txt"), "content2")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "second commit")
	runGit(t, dir, "tag", "v2.0.0")

	prev, err := PreviousTag("v2.0.0")
	if err != nil {
		t.Fatalf("PreviousTag() error = %v", err)
	}
	if prev != "v1.0.0" {
		t.Errorf("PreviousTag() = %q, want %q", prev, "v1.0.0")
	}
}

func TestPreviousTagNoPrevious(t *testing.T) {
	dir := setupGitRepo(t, "v1.0.0")
	chdir(t, dir)

	prev, err := PreviousTag("v1.0.0")
	if err != nil {
		t.Fatalf("PreviousTag() error = %v", err)
	}
	if prev != "" {
		t.Errorf("PreviousTag() = %q, want empty string", prev)
	}
}

func TestLogBetween(t *testing.T) {
	dir := setupGitRepo(t, "v1.0.0")
	chdir(t, dir)

	// Add commits after v1.0.0
	writeFile(t, filepath.Join(dir, "file2.txt"), "content2")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feat: add widget")

	writeFile(t, filepath.Join(dir, "file3.txt"), "content3")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "fix: resolve crash")
	runGit(t, dir, "tag", "v2.0.0")

	logs, err := LogBetween("v1.0.0", "v2.0.0")
	if err != nil {
		t.Fatalf("LogBetween() error = %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("LogBetween() returned %d entries, want 2", len(logs))
	}
	// git log returns newest first
	if logs[0] != "fix: resolve crash" {
		t.Errorf("logs[0] = %q, want %q", logs[0], "fix: resolve crash")
	}
	if logs[1] != "feat: add widget" {
		t.Errorf("logs[1] = %q, want %q", logs[1], "feat: add widget")
	}
}

func TestLogBetweenEmptyFrom(t *testing.T) {
	dir := setupGitRepo(t, "v1.0.0")
	chdir(t, dir)

	logs, err := LogBetween("", "v1.0.0")
	if err != nil {
		t.Fatalf("LogBetween() error = %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("LogBetween() returned %d entries, want 1", len(logs))
	}
	if logs[0] != "initial commit" {
		t.Errorf("logs[0] = %q, want %q", logs[0], "initial commit")
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

func chdir(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(original) })
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
