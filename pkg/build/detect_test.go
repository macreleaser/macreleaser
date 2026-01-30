package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectWorkspace(t *testing.T) {
	tests := []struct {
		name      string
		files     []string // directories to create in temp dir
		wantPath  string
		wantType  WorkspaceType
		wantErr   bool
		errSubstr string
	}{
		{
			name:     "single xcworkspace",
			files:    []string{"MyApp.xcworkspace"},
			wantPath: "MyApp.xcworkspace",
			wantType: Workspace,
		},
		{
			name:     "single xcodeproj",
			files:    []string{"MyApp.xcodeproj"},
			wantPath: "MyApp.xcodeproj",
			wantType: Project,
		},
		{
			name:     "workspace preferred over project",
			files:    []string{"MyApp.xcworkspace", "MyApp.xcodeproj"},
			wantPath: "MyApp.xcworkspace",
			wantType: Workspace,
		},
		{
			name:      "multiple workspaces",
			files:     []string{"App1.xcworkspace", "App2.xcworkspace"},
			wantErr:   true,
			errSubstr: "multiple .xcworkspace files found",
		},
		{
			name:      "multiple projects no workspace",
			files:     []string{"App1.xcodeproj", "App2.xcodeproj"},
			wantErr:   true,
			errSubstr: "multiple .xcodeproj files found",
		},
		{
			name:      "no workspace or project",
			files:     []string{"README.md"},
			wantErr:   true,
			errSubstr: "no .xcworkspace or .xcodeproj found",
		},
		{
			name:     "pods workspace filtered out",
			files:    []string{"MyApp.xcworkspace", "Pods.xcworkspace"},
			wantPath: "MyApp.xcworkspace",
			wantType: Workspace,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			// Create the test files/directories
			for _, f := range tt.files {
				path := filepath.Join(dir, f)
				if err := os.MkdirAll(path, 0755); err != nil {
					t.Fatal(err)
				}
			}

			result, err := DetectWorkspace(dir)
			if tt.wantErr {
				if err == nil {
					t.Fatal("DetectWorkspace() expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errSubstr) {
					t.Errorf("error = %v, want error containing %q", err, tt.errSubstr)
				}
				return
			}

			if err != nil {
				t.Fatalf("DetectWorkspace() unexpected error: %v", err)
			}

			if result.Path != tt.wantPath {
				t.Errorf("Path = %q, want %q", result.Path, tt.wantPath)
			}
			if result.Type != tt.wantType {
				t.Errorf("Type = %v, want %v", result.Type, tt.wantType)
			}
		})
	}
}

func TestDetectWorkspaceEmptyDir(t *testing.T) {
	dir := t.TempDir()
	_, err := DetectWorkspace(dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no .xcworkspace or .xcodeproj found") {
		t.Errorf("error = %v, want 'no .xcworkspace or .xcodeproj found'", err)
	}
}
