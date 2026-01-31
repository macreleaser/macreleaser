package homebrew

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gogithub "github.com/google/go-github/github"
	"github.com/macreleaser/macreleaser/pkg/config"
	macCtx "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/github"
	"github.com/sirupsen/logrus"
)

func newTestContext(t *testing.T) (*macCtx.Context, string) {
	t.Helper()

	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tmpDir := t.TempDir()

	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "TestApp"},
		Release: config.ReleaseConfig{
			GitHub: config.GitHubConfig{
				Owner: "testowner",
				Repo:  "testrepo",
			},
		},
		Homebrew: config.HomebrewConfig{
			Cask: config.CaskConfig{
				Name:     "testapp",
				Desc:     "A test application",
				Homepage: "https://example.com",
			},
		},
	}

	ctx := macCtx.NewContext(context.Background(), cfg, logger)
	ctx.Version = "v1.2.3"
	ctx.Artifacts.BuildOutputDir = tmpDir
	ctx.Artifacts.AppPath = "/path/to/TestApp.app"

	// Create a temp file as a stand-in .zip package
	zipPath := filepath.Join(tmpDir, "TestApp-1.2.3.zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip-content"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx.Artifacts.Packages = []string{zipPath}

	return ctx, tmpDir
}

func TestPipeString(t *testing.T) {
	p := Pipe{}
	expected := "generating Homebrew cask"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestPipeSkipPublish(t *testing.T) {
	ctx, _ := newTestContext(t)
	ctx.SkipPublish = true

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected skip error, got nil")
	}

	var s interface{ IsSkip() bool }
	if !errors.As(err, &s) || !s.IsSkip() {
		t.Errorf("Run() error should satisfy IsSkip, got %T: %v", err, err)
	}

	if !strings.Contains(err.Error(), "homebrew publishing skipped") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "homebrew publishing skipped")
	}
}

func TestPipeNoPackages(t *testing.T) {
	ctx, _ := newTestContext(t)
	ctx.Artifacts.Packages = nil

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected error for empty packages, got nil")
	}

	if !strings.Contains(err.Error(), "no packages found for Homebrew cask") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "no packages found for Homebrew cask")
	}
}

func TestPipeNoAppPath(t *testing.T) {
	ctx, _ := newTestContext(t)
	ctx.Artifacts.AppPath = ""

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected error for missing app path, got nil")
	}

	if !strings.Contains(err.Error(), "no .app path found") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "no .app path found")
	}
}

func TestPipeGenerateCaskAndCommitToTap(t *testing.T) {
	ctx, tmpDir := newTestContext(t)

	// Configure custom tap
	ctx.Config.Homebrew.Tap = config.TapConfig{
		Owner: "tapowner",
		Name:  "homebrew-tap",
		Token: "fake-token",
	}

	mock := github.NewMockClient()
	// Simulate 404 — cask file doesn't exist yet
	mock.ContentsError = fmt.Errorf("file Casks/testapp.rb not found in tapowner/homebrew-tap: 404 Not Found")
	ctx.HomebrewClient = mock

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Verify local cask file was written
	localPath := filepath.Join(tmpDir, "testapp.rb")
	if ctx.Artifacts.HomebrewCaskPath != localPath {
		t.Errorf("HomebrewCaskPath = %q, want %q", ctx.Artifacts.HomebrewCaskPath, localPath)
	}

	content, err := os.ReadFile(localPath)
	if err != nil {
		t.Fatalf("failed to read generated cask file: %v", err)
	}

	cask := string(content)

	// Verify cask content
	expectations := []string{
		`cask "testapp" do`,
		`version "1.2.3"`,
		`url "https://github.com/testowner/testrepo/releases/download/v1.2.3/TestApp-1.2.3.zip"`,
		`name "TestApp"`,
		`desc "A test application"`,
		`homepage "https://example.com"`,
		`app "TestApp.app"`,
	}
	for _, exp := range expectations {
		if !strings.Contains(cask, exp) {
			t.Errorf("cask file missing %q\ngot:\n%s", exp, cask)
		}
	}

	// Verify sha256 is present and non-empty
	if !strings.Contains(cask, `sha256 "`) {
		t.Errorf("cask file missing sha256 stanza\ngot:\n%s", cask)
	}

	// Verify file was created in mock (not updated, since we simulated 404)
	key := "tapowner/homebrew-tap/Casks/testapp.rb"
	if _, exists := mock.CreatedFiles[key]; !exists {
		t.Errorf("expected file to be created at %q in mock, but it wasn't", key)
	}
	if _, exists := mock.UpdatedFiles[key]; exists {
		t.Errorf("file should not have been updated (it was a new file)")
	}
}

func TestPipeUpdateExistingCask(t *testing.T) {
	ctx, _ := newTestContext(t)

	// Configure custom tap
	ctx.Config.Homebrew.Tap = config.TapConfig{
		Owner: "tapowner",
		Name:  "homebrew-tap",
		Token: "fake-token",
	}

	mock := github.NewMockClient()
	// Pre-populate with existing cask file
	sha := "existing-sha-abc123"
	mock.AddFileContent("tapowner", "homebrew-tap", "Casks/testapp.rb", &gogithub.RepositoryContent{
		SHA: &sha,
	})
	ctx.HomebrewClient = mock

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Verify file was updated (not created)
	key := "tapowner/homebrew-tap/Casks/testapp.rb"
	if _, exists := mock.UpdatedFiles[key]; !exists {
		t.Errorf("expected file to be updated at %q in mock, but it wasn't", key)
	}
	if _, exists := mock.CreatedFiles[key]; exists {
		t.Errorf("file should not have been created (it already existed)")
	}
}

func TestPipeNoTapConfigured(t *testing.T) {
	ctx, tmpDir := newTestContext(t)
	// Tap fields are empty by default — no tap commit should happen

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Verify cask file was generated locally
	localPath := filepath.Join(tmpDir, "testapp.rb")
	if ctx.Artifacts.HomebrewCaskPath != localPath {
		t.Errorf("HomebrewCaskPath = %q, want %q", ctx.Artifacts.HomebrewCaskPath, localPath)
	}

	if _, err := os.Stat(localPath); err != nil {
		t.Errorf("local cask file should exist at %q: %v", localPath, err)
	}

	// Verify no HomebrewClient was created (no tap interactions)
	if ctx.HomebrewClient != nil {
		t.Error("HomebrewClient should be nil when no tap is configured")
	}
}

func TestPipeCommitToTapError(t *testing.T) {
	ctx, _ := newTestContext(t)

	ctx.Config.Homebrew.Tap = config.TapConfig{
		Owner: "tapowner",
		Name:  "homebrew-tap",
		Token: "fake-token",
	}

	mock := github.NewMockClient()
	// ContentsError returns 404 to trigger CreateFile path
	mock.ContentsError = fmt.Errorf("404 Not Found")
	// ErrorToReturn will cause CreateFile to fail
	mock.SetError(fmt.Errorf("permission denied"))
	ctx.HomebrewClient = mock

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to commit cask to tap") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "failed to commit cask to tap")
	}
}
