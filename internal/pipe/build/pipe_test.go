package build

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/macreleaser/macreleaser/pkg/config"
	macCtx "github.com/macreleaser/macreleaser/pkg/context"
	"github.com/sirupsen/logrus"
)

func TestPipeOutputDirExists(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a temporary directory structure with pre-existing output dir
	dir := t.TempDir()
	outputDir := filepath.Join(dir, "dist", "TestApp", "v1.0.0")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Change to the temp directory so dist/ is found
	origDir, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(origDir) }()

	cfg := &config.Config{
		Project: config.ProjectConfig{
			Name:   "TestApp",
			Scheme: "TestApp",
		},
		Build: config.BuildConfig{
			Configuration: "Release",
		},
	}
	ctx := macCtx.NewContext(context.Background(), cfg, logger)
	ctx.Version = "v1.0.0"

	err := Pipe{}.Run(ctx)
	if err == nil {
		t.Fatal("expected error for pre-existing output directory, got nil")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("error = %v, want containing 'already exists'", err)
	}
}

func TestPipeString(t *testing.T) {
	p := Pipe{}
	expected := "building project"
	if got := p.String(); got != expected {
		t.Errorf("String() = %q, want %q", got, expected)
	}
}

func TestResolveWorkspace(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	tests := []struct {
		name          string
		workspace     string
		wantWorkspace string
		wantErr       bool
		errSubstr     string
	}{
		{
			name:          "configured xcworkspace",
			workspace:     "MyApp.xcworkspace",
			wantWorkspace: "MyApp.xcworkspace",
		},
		{
			name:          "configured xcodeproj",
			workspace:     "MyApp.xcodeproj",
			wantWorkspace: "MyApp.xcodeproj",
		},
		{
			name:      "invalid extension",
			workspace: "MyApp.invalid",
			wantErr:   true,
			errSubstr: "must end with .xcworkspace or .xcodeproj",
		},
		{
			name:      "path traversal with ..",
			workspace: "../../etc/evil.xcworkspace",
			wantErr:   true,
			errSubstr: "path traversal",
		},
		{
			name:      "absolute path",
			workspace: "/tmp/evil.xcworkspace",
			wantErr:   true,
			errSubstr: "path traversal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Project: config.ProjectConfig{
					Workspace: tt.workspace,
				},
			}
			c := macCtx.NewContext(context.Background(), cfg, logger)

			ws, _, err := resolveWorkspace(c)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error = %v, want containing %q", err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ws != tt.wantWorkspace {
				t.Errorf("workspace = %q, want %q", ws, tt.wantWorkspace)
			}
		})
	}
}

func TestExtractApp(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a fake .xcarchive structure
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "MyApp.xcarchive")
	appsDir := filepath.Join(archivePath, "Products", "Applications", "MyApp.app", "Contents")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Write a file inside the .app to make it non-empty
	if err := os.WriteFile(filepath.Join(appsDir, "Info.plist"), []byte("<plist/>"), 0644); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(dir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	c := macCtx.NewContext(context.Background(), cfg, logger)

	err := extractApp(c, archivePath, outputDir)
	if err != nil {
		t.Fatalf("extractApp() error = %v", err)
	}

	expectedAppPath := filepath.Join(outputDir, "MyApp.app")
	if c.Artifacts.AppPath != expectedAppPath {
		t.Errorf("AppPath = %q, want %q", c.Artifacts.AppPath, expectedAppPath)
	}

	// Verify the .app was actually copied
	info, err := os.Stat(expectedAppPath)
	if err != nil {
		t.Fatalf("copied .app does not exist: %v", err)
	}
	if !info.IsDir() {
		t.Error("copied .app is not a directory")
	}
}

func TestExtractAppNoApp(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Create a fake .xcarchive without any .app
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "MyApp.xcarchive")
	appsDir := filepath.Join(archivePath, "Products", "Applications")
	if err := os.MkdirAll(appsDir, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	c := macCtx.NewContext(context.Background(), cfg, logger)

	err := extractApp(c, archivePath, dir)
	if err == nil {
		t.Fatal("expected error for missing .app")
	}
	if !strings.Contains(err.Error(), "no .app found") {
		t.Errorf("error = %v, want containing 'no .app found'", err)
	}
}

func TestExtractAppMissingProductsDir(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	dir := t.TempDir()
	archivePath := filepath.Join(dir, "MyApp.xcarchive")
	if err := os.MkdirAll(archivePath, 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &config.Config{}
	c := macCtx.NewContext(context.Background(), cfg, logger)

	err := extractApp(c, archivePath, dir)
	if err == nil {
		t.Fatal("expected error for missing Products/Applications")
	}
	if !strings.Contains(err.Error(), "Products/Applications") {
		t.Errorf("error = %v, want containing 'Products/Applications'", err)
	}
}
