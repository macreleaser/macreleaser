package homebrew

import (
	"strings"
	"testing"
)

func TestRenderCask(t *testing.T) {
	data := CaskData{
		Token:    "myapp",
		Version:  "1.2.3",
		SHA256:   "abc123def456",
		URL:      "https://github.com/owner/repo/releases/download/v1.2.3/MyApp-1.2.3.zip",
		Name:     "MyApp",
		Desc:     "A great macOS application",
		Homepage: "https://example.com",
		AppName:  "MyApp.app",
	}

	got, err := RenderCask(data)
	if err != nil {
		t.Fatalf("RenderCask() unexpected error: %v", err)
	}

	// Verify all expected stanzas are present
	expectations := []string{
		`cask "myapp" do`,
		`version "1.2.3"`,
		`sha256 "abc123def456"`,
		`url "https://github.com/owner/repo/releases/download/v1.2.3/MyApp-1.2.3.zip"`,
		`name "MyApp"`,
		`desc "A great macOS application"`,
		`homepage "https://example.com"`,
		`app "MyApp.app"`,
		"end",
	}

	for _, exp := range expectations {
		if !strings.Contains(got, exp) {
			t.Errorf("RenderCask() output missing %q\ngot:\n%s", exp, got)
		}
	}
}

func TestRenderCaskWithLicense(t *testing.T) {
	data := CaskData{
		Token:    "myapp",
		Version:  "1.0.0",
		SHA256:   "abc123",
		URL:      "https://example.com/myapp.zip",
		Name:     "MyApp",
		Desc:     "An app",
		Homepage: "https://example.com",
		AppName:  "MyApp.app",
		License:  "mit",
	}

	got, err := RenderCask(data)
	if err != nil {
		t.Fatalf("RenderCask() unexpected error: %v", err)
	}

	if !strings.Contains(got, `license "mit"`) {
		t.Errorf("RenderCask() output missing license stanza\ngot:\n%s", got)
	}
}

func TestRenderCaskWithoutLicense(t *testing.T) {
	data := CaskData{
		Token:    "myapp",
		Version:  "1.0.0",
		SHA256:   "abc123",
		URL:      "https://example.com/myapp.zip",
		Name:     "MyApp",
		Desc:     "An app",
		Homepage: "https://example.com",
		AppName:  "MyApp.app",
		License:  "",
	}

	got, err := RenderCask(data)
	if err != nil {
		t.Fatalf("RenderCask() unexpected error: %v", err)
	}

	if strings.Contains(got, "license") {
		t.Errorf("RenderCask() output should not contain license stanza when License is empty\ngot:\n%s", got)
	}
}

func TestBuildAssetURL(t *testing.T) {
	tests := []struct {
		owner, repo, tag, filename string
		want                       string
	}{
		{
			owner: "owner", repo: "repo", tag: "v1.2.3", filename: "App-1.2.3.zip",
			want: "https://github.com/owner/repo/releases/download/v1.2.3/App-1.2.3.zip",
		},
		{
			owner: "myorg", repo: "myapp", tag: "v0.1.0", filename: "MyApp-0.1.0.dmg",
			want: "https://github.com/myorg/myapp/releases/download/v0.1.0/MyApp-0.1.0.dmg",
		},
	}

	for _, tt := range tests {
		got := BuildAssetURL(tt.owner, tt.repo, tt.tag, tt.filename)
		if got != tt.want {
			t.Errorf("BuildAssetURL(%q, %q, %q, %q) = %q, want %q",
				tt.owner, tt.repo, tt.tag, tt.filename, got, tt.want)
		}
	}
}

func TestSelectPackage(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		wantExt  string
		wantErr  bool
	}{
		{
			name:     "prefer zip over dmg",
			packages: []string{"/path/to/App.zip", "/path/to/App.dmg"},
			wantExt:  ".zip",
		},
		{
			name:     "zip only",
			packages: []string{"/path/to/App.zip"},
			wantExt:  ".zip",
		},
		{
			name:     "dmg only",
			packages: []string{"/path/to/App.dmg"},
			wantExt:  ".dmg",
		},
		{
			name:     "dmg before zip still selects zip",
			packages: []string{"/path/to/App.dmg", "/path/to/App.zip"},
			wantExt:  ".zip",
		},
		{
			name:     "no zip or dmg",
			packages: []string{"/path/to/App.app"},
			wantErr:  true,
		},
		{
			name:     "empty list",
			packages: []string{},
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SelectPackage(tt.packages)
			if (err != nil) != tt.wantErr {
				t.Fatalf("SelectPackage() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if !strings.Contains(err.Error(), "no .zip or .dmg") {
					t.Errorf("SelectPackage() error = %q, want error containing %q", err.Error(), "no .zip or .dmg")
				}
				return
			}
			if !strings.HasSuffix(got, tt.wantExt) {
				t.Errorf("SelectPackage() = %q, want file with extension %q", got, tt.wantExt)
			}
		})
	}
}
