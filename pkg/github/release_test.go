package github

import "testing"

func TestContentTypeForAsset(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "zip file", path: "dist/MyApp-1.0.0.zip", want: "application/zip"},
		{name: "dmg file", path: "dist/MyApp-1.0.0.dmg", want: "application/x-apple-diskimage"},
		{name: "pkg file", path: "dist/MyApp-1.0.0.pkg", want: "application/octet-stream"},
		{name: "tar.gz file", path: "dist/MyApp-1.0.0.tar.gz", want: "application/octet-stream"},
		{name: "empty string", path: "", want: "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ContentTypeForAsset(tt.path)
			if got != tt.want {
				t.Errorf("ContentTypeForAsset(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}
