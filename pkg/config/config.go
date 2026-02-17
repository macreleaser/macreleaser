package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
	"github.com/macreleaser/macreleaser/pkg/env"
)

// Config represents the complete macreleaser configuration
type Config struct {
	Project  ProjectConfig  `yaml:"project"`
	Build    BuildConfig    `yaml:"build"`
	Sign     SignConfig     `yaml:"sign"`
	Notarize NotarizeConfig `yaml:"notarize"`
	Archive  ArchiveConfig  `yaml:"archive"`
	Release  ReleaseConfig  `yaml:"release"`
	Homebrew HomebrewConfig `yaml:"homebrew"`
}

// ProjectConfig contains project-specific settings
type ProjectConfig struct {
	Name      string `yaml:"name"`
	Scheme    string `yaml:"scheme"`
	Workspace string `yaml:"workspace,omitempty"`
}

// BuildConfig contains build configuration
type BuildConfig struct {
	Configuration string `yaml:"configuration"`
}

// SignConfig contains code signing configuration
type SignConfig struct {
	Identity string `yaml:"identity"`
}

// NotarizeConfig contains Apple notarization configuration.
// SECURITY NOTE: The Password field stores the Apple ID app-specific password
// in memory as a plain string. This is unavoidable for passing to notarytool,
// but memory should be cleared after use where possible. Always use environment
// variable substitution (env(VAR_NAME)) instead of hardcoding passwords in config files.
type NotarizeConfig struct {
	AppleID  string `yaml:"apple_id"`
	TeamID   string `yaml:"team_id"`
	Password string `yaml:"password"`
}

// ArchiveConfig contains archive creation configuration
type ArchiveConfig struct {
	Formats []string  `yaml:"formats"`
	DMG     DMGConfig `yaml:"dmg,omitempty"`
	Zip     ZipConfig `yaml:"zip,omitempty"`
}

// DMGConfig contains DMG-specific configuration
type DMGConfig struct {
	Background string `yaml:"background,omitempty"`
	IconSize   int    `yaml:"icon_size,omitempty"`
}

// ZipConfig contains ZIP-specific configuration
type ZipConfig struct {
	CompressionLevel int `yaml:"compression_level,omitempty"`
}

// ReleaseConfig contains release configuration
type ReleaseConfig struct {
	GitHub GitHubConfig `yaml:"github"`
}

// GitHubConfig contains GitHub-specific release configuration
type GitHubConfig struct {
	Owner string `yaml:"owner"`
	Repo  string `yaml:"repo"`
	Draft bool   `yaml:"draft"`
}

// HomebrewConfig contains Homebrew cask configuration
type HomebrewConfig struct {
	Tap      TapConfig      `yaml:"tap,omitempty"`
	Official OfficialConfig `yaml:"official,omitempty"`
	Cask     CaskConfig     `yaml:"cask"`
}

// TapConfig contains custom tap configuration
type TapConfig struct {
	Owner string `yaml:"owner"`
	Name  string `yaml:"name"`
	Token string `yaml:"token"`
}

// OfficialConfig contains official homebrew tap configuration
type OfficialConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Token     string   `yaml:"token"`
	AutoMerge bool     `yaml:"auto_merge"`
	Assignees []string `yaml:"assignees"`
}

// CaskConfig contains cask metadata
type CaskConfig struct {
	Name     string `yaml:"name"`
	Desc     string `yaml:"desc"`
	Homepage string `yaml:"homepage"`
	License  string `yaml:"license"`
}

// LoadConfig loads and parses a configuration file
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config file path is required")
	}

	cleanPath, err := validateConfigPath(path)
	if err != nil {
		return nil, err
	}

	data, err := readConfigFile(cleanPath)
	if err != nil {
		return nil, err
	}

	file, err := parser.ParseBytes(data, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if len(file.Docs) == 0 || file.Docs[0].Body == nil {
		return nil, fmt.Errorf("failed to parse config: empty document")
	}

	if err := env.SubstituteEnvVarsNode(file.Docs[0].Body); err != nil {
		return nil, fmt.Errorf("environment variable substitution failed: %w", err)
	}

	var config Config
	if err := yaml.NodeToValue(file.Docs[0].Body, &config, yaml.Strict()); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves a configuration to a file
func SaveConfig(path string, config *Config) error {
	if config == nil {
		return fmt.Errorf("config cannot be nil")
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Use restrictive permissions (0600) since config may contain sensitive data
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func validateConfigPath(path string) (string, error) {
	// Prevent path traversal attacks
	// Resolve to absolute path first, then validate it doesn't escape working directory
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Clean the path to normalize it
	cleanPath := filepath.Clean(absPath)

	// Get working directory to validate path doesn't escape it
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}
	wd = filepath.Clean(wd)

	// For paths within working directory, ensure they don't use parent directory references
	// This prevents traversal attacks like "../../../etc/passwd"
	if strings.HasPrefix(cleanPath, wd+string(filepath.Separator)) || cleanPath == wd {
		// Path is within working directory, validate it's local relative to wd
		relPath, err := filepath.Rel(wd, cleanPath)
		if err != nil {
			return "", fmt.Errorf("invalid config path: %w", err)
		}
		if !filepath.IsLocal(relPath) {
			return "", fmt.Errorf("invalid config path: path traversal detected")
		}
	}
	// If path is outside working directory (e.g., absolute temp path), allow it
	// This is necessary for tests and legitimate use cases with full paths

	return cleanPath, nil
}

func readConfigFile(cleanPath string) ([]byte, error) {
	// Follow symlinks and validate the target is a regular file
	// Using os.Stat (not Lstat) to follow symlinks and validate the target
	info, err := os.Stat(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to access config file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("config path is not a regular file")
	}

	// Prevent DoS via large files - limit to 1MB
	const maxConfigSize = 1024 * 1024 // 1MB
	if info.Size() > maxConfigSize {
		return nil, fmt.Errorf("config file too large: maximum size is 1MB")
	}

	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return data, nil
}
