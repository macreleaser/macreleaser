package homebrew

import (
	"bytes"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"
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

// validateCaskField checks that a string value is safe for embedding in a
// Homebrew cask Ruby file. Double quotes, backslashes, and newlines would
// break Ruby string literals. Ruby interpolation (#{}) inside double-quoted
// strings executes arbitrary code at parse time.
func validateCaskField(name, value string) error {
	if strings.ContainsAny(value, "\"\n\r\\") {
		return fmt.Errorf("invalid %s: must not contain double quotes, backslashes, or newlines", name)
	}
	if strings.Contains(value, "#{") {
		return fmt.Errorf("invalid %s: must not contain Ruby interpolation sequences", name)
	}
	return nil
}

// RenderCask renders a Homebrew cask Ruby file from the given data.
func RenderCask(data CaskData) (string, error) {
	fields := map[string]string{
		"token":    data.Token,
		"version":  data.Version,
		"url":      data.URL,
		"name":     data.Name,
		"desc":     data.Desc,
		"homepage": data.Homepage,
		"app_name": data.AppName,
		"license":  data.License,
	}
	for name, value := range fields {
		if err := validateCaskField(name, value); err != nil {
			return "", err
		}
	}
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
		owner, repo, tag, url.PathEscape(filename))
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
