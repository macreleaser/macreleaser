package github

import "path/filepath"

// ContentTypeForAsset returns the MIME content type for a release asset
// based on its file extension. Unknown extensions default to application/octet-stream.
func ContentTypeForAsset(path string) string {
	switch filepath.Ext(path) {
	case ".zip":
		return "application/zip"
	case ".dmg":
		return "application/x-apple-diskimage"
	default:
		return "application/octet-stream"
	}
}
