package homebrew

import (
	"bytes"
	"fmt"
	"path/filepath"
	"text/template"
)

// CaskData contains all fields needed to render a Homebrew cask file.
type CaskData struct {
	Token    string // cask token/identifier (e.g., "myapp")
	Version  string // bare version without v prefix (e.g., "1.2.3")
	SHA256   string // hex-encoded SHA256 hash
	URL      string // direct download URL for the archive
	Name     string // human-readable app name (e.g., "MyApp")
	Desc     string // short description
	Homepage string // homepage URL
	AppName  string // .app bundle name (e.g., "MyApp.app")
	License  string // optional SPDX license identifier
}

const caskTemplate = `cask "{{.Token}}" do
  version "{{.Version}}"
  sha256 "{{.SHA256}}"

  url "{{.URL}}"
  name "{{.Name}}"
  desc "{{.Desc}}"
  homepage "{{.Homepage}}"
{{- if .License}}

  license "{{.License}}"
{{- end}}

  app "{{.AppName}}"
end
`

// RenderCask renders a Homebrew cask Ruby file from the given data.
func RenderCask(data CaskData) (string, error) {
	tmpl, err := template.New("cask").Parse(caskTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse cask template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to render cask template: %w", err)
	}

	return buf.String(), nil
}

// BuildAssetURL constructs the GitHub release asset download URL.
func BuildAssetURL(owner, repo, tag, filename string) string {
	return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
		owner, repo, tag, filename)
}

// SelectPackage selects the preferred archive from the package list for
// use in the Homebrew cask. Prefers .zip, falls back to .dmg.
func SelectPackage(packages []string) (string, error) {
	for _, p := range packages {
		if filepath.Ext(p) == ".zip" {
			return p, nil
		}
	}
	for _, p := range packages {
		if filepath.Ext(p) == ".dmg" {
			return p, nil
		}
	}
	return "", fmt.Errorf("no .zip or .dmg package found for Homebrew cask — ensure archive formats include zip or dmg")
}
