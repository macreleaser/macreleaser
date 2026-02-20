package build

import (
	"testing"
)

func TestBuildArchiveArgs(t *testing.T) {
	tests := []struct {
		name string
		args XcodebuildArgs
		want []string
	}{
		{
			name: "workspace with all options",
			args: XcodebuildArgs{
				Scheme:        "MyApp",
				Workspace:     "MyApp.xcworkspace",
				WorkspaceType: Workspace,
				Configuration: "Release",
				ArchivePath:   "dist/MyApp/v1.0.0/MyApp.xcarchive",
			},
			want: []string{
				"-workspace", "MyApp.xcworkspace",
				"-scheme", "MyApp",
				"-configuration", "Release",
				"-archivePath", "dist/MyApp/v1.0.0/MyApp.xcarchive",
				"archive",
				"CODE_SIGN_IDENTITY=-",
			},
		},
		{
			name: "project instead of workspace",
			args: XcodebuildArgs{
				Scheme:        "MyApp",
				Workspace:     "MyApp.xcodeproj",
				WorkspaceType: Project,
				Configuration: "Release",
				ArchivePath:   "dist/MyApp/v1.0.0/MyApp.xcarchive",
			},
			want: []string{
				"-project", "MyApp.xcodeproj",
				"-scheme", "MyApp",
				"-configuration", "Release",
				"-archivePath", "dist/MyApp/v1.0.0/MyApp.xcarchive",
				"archive",
				"CODE_SIGN_IDENTITY=-",
			},
		},
		{
			name: "minimal args",
			args: XcodebuildArgs{
				Scheme: "MyApp",
			},
			want: []string{
				"-scheme", "MyApp",
				"archive",
				"CODE_SIGN_IDENTITY=-",
			},
		},
		{
			name: "empty args",
			args: XcodebuildArgs{},
			want: []string{"archive", "CODE_SIGN_IDENTITY=-"},
		},
		{
			name: "with version injection",
			args: XcodebuildArgs{
				Scheme:        "MyApp",
				Workspace:     "MyApp.xcworkspace",
				WorkspaceType: Workspace,
				Configuration: "Release",
				ArchivePath:   "dist/MyApp.xcarchive",
				Version:       "1.2.3",
				BuildNumber:   "42",
			},
			want: []string{
				"-workspace", "MyApp.xcworkspace",
				"-scheme", "MyApp",
				"-configuration", "Release",
				"-archivePath", "dist/MyApp.xcarchive",
				"archive",
				"CODE_SIGN_IDENTITY=-",
				"MARKETING_VERSION=1.2.3",
				"CURRENT_PROJECT_VERSION=42",
			},
		},
		{
			name: "version only without build number",
			args: XcodebuildArgs{
				Scheme:  "MyApp",
				Version: "2.0.0",
			},
			want: []string{
				"-scheme", "MyApp",
				"archive",
				"CODE_SIGN_IDENTITY=-",
				"MARKETING_VERSION=2.0.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildArchiveArgs(tt.args)

			if len(got) != len(tt.want) {
				t.Fatalf("BuildArchiveArgs() returned %d args, want %d\ngot:  %v\nwant: %v", len(got), len(tt.want), got, tt.want)
			}

			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("arg[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
