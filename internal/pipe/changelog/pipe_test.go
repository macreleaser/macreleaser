package changelog

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
	macCtx "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/git"
	"github.com/sirupsen/logrus"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := tryRunGit(dir, "init", "--template="); err != nil {
		t.Skipf("Skipping: git init not available: %v", err)
	}

	runGit(t, dir, "config", "user.email", "test@test.com")
	runGit(t, dir, "config", "user.name", "Test")

	writeFile(t, filepath.Join(dir, "file.txt"), "content")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "initial commit")
	runGit(t, dir, "tag", "v1.0.0")

	writeFile(t, filepath.Join(dir, "file2.txt"), "content2")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "feat: add new feature")

	writeFile(t, filepath.Join(dir, "file3.txt"), "content3")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "fix: resolve crash")
	runGit(t, dir, "tag", "v2.0.0")

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

func TestPipeRun(t *testing.T) {
	dir := setupGitRepo(t)
	chdir(t, dir)

	distDir := filepath.Join(dir, "dist")
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	ctx := macCtx.NewContext(context.Background(), &config.Config{}, logger)
	ctx.Version = "v2.0.0"
	ctx.Git = git.GitInfo{Tag: "v2.0.0"}
	ctx.Artifacts.BuildOutputDir = distDir

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if ctx.ReleaseNotes == "" {
		t.Error("ReleaseNotes should not be empty")
	}
	if !strings.Contains(ctx.ReleaseNotes, "## v2.0.0") {
		t.Errorf("ReleaseNotes missing version heading:\n%s", ctx.ReleaseNotes)
	}
	if !strings.Contains(ctx.ReleaseNotes, "feat: add new feature") {
		t.Errorf("ReleaseNotes missing feat commit:\n%s", ctx.ReleaseNotes)
	}
	if !strings.Contains(ctx.ReleaseNotes, "fix: resolve crash") {
		t.Errorf("ReleaseNotes missing fix commit:\n%s", ctx.ReleaseNotes)
	}

	// Verify file was written
	changelogPath := filepath.Join(distDir, "CHANGELOG.md")
	if ctx.Artifacts.ChangelogPath != changelogPath {
		t.Errorf("ChangelogPath = %q, want %q", ctx.Artifacts.ChangelogPath, changelogPath)
	}

	data, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("failed to read CHANGELOG.md: %v", err)
	}
	if string(data) != ctx.ReleaseNotes {
		t.Error("CHANGELOG.md content doesn't match ReleaseNotes")
	}
}

func TestPipeRunWithGroups(t *testing.T) {
	dir := setupGitRepo(t)
	chdir(t, dir)

	distDir := filepath.Join(dir, "dist")
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	ctx := macCtx.NewContext(context.Background(), &config.Config{
		Changelog: config.ChangelogConfig{
			Groups: []config.ChangelogGroupConfig{
				{Title: "Features", Regexp: "^feat:", Order: 0},
				{Title: "Bug Fixes", Regexp: "^fix:", Order: 1},
			},
		},
	}, logger)
	ctx.Version = "v2.0.0"
	ctx.Git = git.GitInfo{Tag: "v2.0.0"}
	ctx.Artifacts.BuildOutputDir = distDir

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if !strings.Contains(ctx.ReleaseNotes, "### Features") {
		t.Error("ReleaseNotes missing Features group")
	}
	if !strings.Contains(ctx.ReleaseNotes, "### Bug Fixes") {
		t.Error("ReleaseNotes missing Bug Fixes group")
	}
}

func TestPipeRunDisabled(t *testing.T) {
	dir := setupGitRepo(t)
	chdir(t, dir)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	ctx := macCtx.NewContext(context.Background(), &config.Config{
		Changelog: config.ChangelogConfig{Disable: true},
	}, logger)
	ctx.Version = "v2.0.0"

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected skip error, got nil")
	}
	var s interface{ IsSkip() bool }
	if !errors.As(err, &s) || !s.IsSkip() {
		t.Errorf("expected skip error, got %T: %v", err, err)
	}
}

func TestPipeRunDefaultDistDir(t *testing.T) {
	dir := setupGitRepo(t)
	chdir(t, dir)

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	ctx := macCtx.NewContext(context.Background(), &config.Config{}, logger)
	ctx.Version = "v2.0.0"
	ctx.Git = git.GitInfo{Tag: "v2.0.0"}

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	expectedPath := filepath.Join("dist", "CHANGELOG.md")
	if ctx.Artifacts.ChangelogPath != expectedPath {
		t.Errorf("ChangelogPath = %q, want %q", ctx.Artifacts.ChangelogPath, expectedPath)
	}

	// Clean up dist/ created in CWD
	t.Cleanup(func() { os.RemoveAll("dist") })
}

func TestPipeString(t *testing.T) {
	p := Pipe{}
	expected := "generating changelog"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}
