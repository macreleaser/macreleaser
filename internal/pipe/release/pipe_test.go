package release

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
	macCtx "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/github"
	"github.com/sirupsen/logrus"
)

func newContext() *macCtx.Context {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	cfg := &config.Config{
		Project: config.ProjectConfig{Name: "TestApp"},
		Release: config.ReleaseConfig{
			GitHub: config.GitHubConfig{
				Owner: "testowner",
				Repo:  "testrepo",
			},
		},
	}
	return macCtx.NewContext(context.Background(), cfg, logger)
}

func TestPipeString(t *testing.T) {
	p := Pipe{}
	expected := "publishing GitHub release"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestPipeSkipPublish(t *testing.T) {
	ctx := newContext()
	ctx.SkipPublish = true

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected skip error, got nil")
	}

	var s interface{ IsSkip() bool }
	if !errors.As(err, &s) || !s.IsSkip() {
		t.Errorf("Run() error should satisfy IsSkip, got %T: %v", err, err)
	}

	if !strings.Contains(err.Error(), "publishing skipped") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "publishing skipped")
	}
}

func TestPipeNoPackages(t *testing.T) {
	ctx := newContext()
	ctx.GitHubClient = github.NewMockClient()

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected error for empty packages, got nil")
	}

	if !strings.Contains(err.Error(), "no packages to release") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "no packages to release")
	}
}

func TestPipeCreateReleaseAndUpload(t *testing.T) {
	ctx := newContext()
	ctx.Version = "v1.2.3"

	mock := github.NewMockClient()
	ctx.GitHubClient = mock

	// Create temporary files as stand-in packages
	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "TestApp-v1.2.3.zip")
	dmgPath := filepath.Join(tmpDir, "TestApp-v1.2.3.dmg")
	if err := os.WriteFile(zipPath, []byte("fake-zip"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dmgPath, []byte("fake-dmg"), 0644); err != nil {
		t.Fatal(err)
	}

	ctx.Artifacts.Packages = []string{zipPath, dmgPath}

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Verify release was created
	key := "testowner/testrepo"
	releases := mock.Releases[key]
	if len(releases) != 1 {
		t.Fatalf("expected 1 release, got %d", len(releases))
	}

	rel := releases[0]
	if got := rel.GetTagName(); got != "v1.2.3" {
		t.Errorf("release tag = %q, want %q", got, "v1.2.3")
	}
	if got := rel.GetName(); got != "TestApp v1.2.3" {
		t.Errorf("release name = %q, want %q", got, "TestApp v1.2.3")
	}
	if rel.GetDraft() {
		t.Error("release draft = true, want false")
	}

	// Verify assets were uploaded
	if len(mock.UploadedAssets) != 2 {
		t.Fatalf("expected 2 uploaded assets, got %d", len(mock.UploadedAssets))
	}
	if mock.UploadedAssets[0] != zipPath {
		t.Errorf("uploaded asset[0] = %q, want %q", mock.UploadedAssets[0], zipPath)
	}
	if mock.UploadedAssets[1] != dmgPath {
		t.Errorf("uploaded asset[1] = %q, want %q", mock.UploadedAssets[1], dmgPath)
	}

	// Verify release URL was set
	expectedURL := "https://github.com/testowner/testrepo/releases/tag/v1.2.3"
	if ctx.Artifacts.ReleaseURL != expectedURL {
		t.Errorf("ReleaseURL = %q, want %q", ctx.Artifacts.ReleaseURL, expectedURL)
	}
}

func TestPipeCreateReleaseDraft(t *testing.T) {
	ctx := newContext()
	ctx.Version = "v2.0.0"
	ctx.Config.Release.GitHub.Draft = true

	mock := github.NewMockClient()
	ctx.GitHubClient = mock

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "TestApp-v2.0.0.zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx.Artifacts.Packages = []string{zipPath}

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	rel := mock.Releases["testowner/testrepo"][0]
	if !rel.GetDraft() {
		t.Error("release draft = false, want true")
	}
}

func TestPipeCreateReleaseError(t *testing.T) {
	ctx := newContext()
	ctx.Version = "v1.0.0"

	mock := github.NewMockClient()
	mock.SetError(fmt.Errorf("API error"))
	ctx.GitHubClient = mock

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "TestApp-v1.0.0.zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx.Artifacts.Packages = []string{zipPath}

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to create GitHub release") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "failed to create GitHub release")
	}
}

func TestPipeCreateReleaseAlreadyExists(t *testing.T) {
	ctx := newContext()
	ctx.Version = "v1.0.0"

	mock := github.NewMockClient()
	mock.SetError(fmt.Errorf("Validation Failed: already_exists"))
	ctx.GitHubClient = mock

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "TestApp-v1.0.0.zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx.Artifacts.Packages = []string{zipPath}

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "already exists")
	}
	if !strings.Contains(err.Error(), "v1.0.0") {
		t.Errorf("Run() error = %q, want error containing version %q", err.Error(), "v1.0.0")
	}
}

func TestPipeUploadAssetError(t *testing.T) {
	ctx := newContext()
	ctx.Version = "v1.0.0"

	mock := github.NewMockClient()
	mock.UploadError = fmt.Errorf("upload failed: connection reset")
	ctx.GitHubClient = mock

	tmpDir := t.TempDir()
	zipPath := filepath.Join(tmpDir, "TestApp-v1.0.0.zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip"), 0644); err != nil {
		t.Fatal(err)
	}
	ctx.Artifacts.Packages = []string{zipPath}

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("Run() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "failed to upload asset") {
		t.Errorf("Run() error = %q, want error containing %q", err.Error(), "failed to upload asset")
	}
	if !strings.Contains(err.Error(), "TestApp-v1.0.0.zip") {
		t.Errorf("Run() error = %q, want error containing filename", err.Error())
	}
}

func TestPipeSkipsNonRegularFiles(t *testing.T) {
	ctx := newContext()
	ctx.Version = "v1.0.0"

	mock := github.NewMockClient()
	ctx.GitHubClient = mock

	tmpDir := t.TempDir()

	// Create a real file
	zipPath := filepath.Join(tmpDir, "TestApp-v1.0.0.zip")
	if err := os.WriteFile(zipPath, []byte("fake-zip"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a directory (simulates .app bundle in Packages)
	appDir := filepath.Join(tmpDir, "TestApp.app")
	if err := os.Mkdir(appDir, 0755); err != nil {
		t.Fatal(err)
	}

	ctx.Artifacts.Packages = []string{appDir, zipPath}

	err := Pipe{}.Run(ctx)
	if err != nil {
		t.Fatalf("Run() unexpected error: %v", err)
	}

	// Only the regular file should have been uploaded
	if len(mock.UploadedAssets) != 1 {
		t.Fatalf("expected 1 uploaded asset, got %d", len(mock.UploadedAssets))
	}
	if mock.UploadedAssets[0] != zipPath {
		t.Errorf("uploaded asset = %q, want %q", mock.UploadedAssets[0], zipPath)
	}
}
