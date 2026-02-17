package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name          string
		yamlContent   string
		expectError   bool
		expectedError string
	}{
		{
			name: "valid minimal config",
			yamlContent: `
project:
  name: "MyApp"
  scheme: "MyApp"
build:
  configuration: "Release"

sign:
  identity: "[TEST_IDENTITY_PLACEHOLDER]"
notarize:
  apple_id: "[TEST_EMAIL_PLACEHOLDER]"
  team_id: "[TEST_TEAM_ID]"
  password: "[TEST_PASSWORD_PLACEHOLDER]"
archive:
  formats: ["dmg"]
release:
  github:
    owner: "testowner"
    repo: "testrepo"
    draft: false
homebrew:
  cask:
    name: "testapp"
    desc: "Test application"
    homepage: "https://example.com"
    license: "MIT"
`,
			expectError: false,
		},
		{
			name: "partial config loads successfully",
			yamlContent: `
project:
  name: "MyApp"
build:
  configuration: "Release"

`,
			expectError: false,
		},
		{
			name: "invalid YAML",
			yamlContent: `
project:
  name: "MyApp"
  invalid_yaml: [unclosed array
`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary config file
			tmpFile := filepath.Join(t.TempDir(), "config.yaml")
			if err := os.WriteFile(tmpFile, []byte(tt.yamlContent), 0644); err != nil {
				t.Fatalf("Failed to create temporary config file: %v", err)
			}

			// Test LoadConfig
			config, err := LoadConfig(tmpFile)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if config == nil {
				t.Error("Expected config but got nil")
				return
			}

			// Verify basic fields are populated
			if config.Project.Name == "" {
				t.Error("Project name should not be empty")
			}
			if config.Build.Configuration == "" {
				t.Error("Build configuration should not be empty")
			}
		})
	}
}

func TestSaveConfig(t *testing.T) {
	config := &Config{
		Project: ProjectConfig{
			Name:   "TestApp",
			Scheme: "TestApp",
		},
		Build: BuildConfig{
			Configuration: "Release",

		},
		Sign: SignConfig{
			Identity: "[TEST_IDENTITY_PLACEHOLDER]",
		},
		Notarize: NotarizeConfig{
			AppleID:  "[TEST_EMAIL_PLACEHOLDER]",
			TeamID:   "[TEST_TEAM_ID]",
			Password: "[TEST_PASSWORD_PLACEHOLDER]",
		},
		Archive: ArchiveConfig{
			Formats: []string{"dmg"},
		},
		Release: ReleaseConfig{
			GitHub: GitHubConfig{
				Owner: "testowner",
				Repo:  "testrepo",
				Draft: false,
			},
		},
		Homebrew: HomebrewConfig{
			Cask: CaskConfig{
				Name:     "testapp",
				Desc:     "Test application",
				Homepage: "https://example.com",
				License:  "MIT",
			},
		},
	}

	tmpFile := filepath.Join(t.TempDir(), "saved-config.yaml")

	// Save config
	err := SaveConfig(tmpFile, config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Load and verify
	loadedConfig, err := LoadConfig(tmpFile)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Project.Name != config.Project.Name {
		t.Errorf("Expected project name %s, got %s", config.Project.Name, loadedConfig.Project.Name)
	}
	if loadedConfig.Build.Configuration != config.Build.Configuration {
		t.Errorf("Expected build configuration %s, got %s", config.Build.Configuration, loadedConfig.Build.Configuration)
	}
}

func TestEnvironmentVariableSubstitution(t *testing.T) {
	// Set test environment variable
	if err := os.Setenv("TEST_IDENTITY", "Developer ID Application: Test (1234567890)"); err != nil {
		t.Fatalf("Failed to set TEST_IDENTITY: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("TEST_IDENTITY")
	}()

	yamlContent := `
project:
  name: "MyApp"
  scheme: "MyApp"
build:
  configuration: "Release"

sign:
  identity: "env(TEST_IDENTITY)"
notarize:
  apple_id: "env(TEST_EMAIL)"
  team_id: "1234567890"
  password: "test-password"
archive:
  formats: ["dmg"]
release:
  github:
    owner: "testowner"
    repo: "testrepo"
    draft: false
homebrew:
  cask:
    name: "testapp"
    desc: "Test application"
    homepage: "https://example.com"
    license: "MIT"
`

	// Set test environment for email
	if err := os.Setenv("TEST_EMAIL", "test@example.com"); err != nil {
		t.Fatalf("Failed to set TEST_EMAIL: %v", err)
	}
	defer func() {
		_ = os.Unsetenv("TEST_EMAIL")
	}()

	tmpFile := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(tmpFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create temporary config file: %v", err)
	}

	config, err := LoadConfig(tmpFile)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	if config.Sign.Identity != "Developer ID Application: Test (1234567890)" {
		t.Errorf("Expected substituted identity, got %s", config.Sign.Identity)
	}

	if config.Notarize.AppleID != "test@example.com" {
		t.Errorf("Expected substituted apple ID, got %s", config.Notarize.AppleID)
	}
}

func TestLoadConfigSecurity(t *testing.T) {
	tests := []struct {
		name          string
		setupFunc     func(t *testing.T) string
		expectError   bool
		errorContains string
	}{
		{
			name: "path traversal attempt via parent directory",
			setupFunc: func(t *testing.T) string {
				// Create a temporary directory structure
				tmpDir := t.TempDir()
				configDir := filepath.Join(tmpDir, "config")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatalf("Failed to create config directory: %v", err)
				}

				// Return path with traversal attempt - this will fail because
				// the path escapes the working directory and the file doesn't exist
				return configDir + "/../../etc/passwd"
			},
			expectError:   true,
			errorContains: "no such file or directory",
		},
		{
			name: "symlink to valid config is allowed",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()

				// Create a valid config file
				configFile := filepath.Join(tmpDir, "real-config.yaml")
				if err := os.WriteFile(configFile, []byte("project:\n  name: TestApp\n"), 0644); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}

				// Create a symlink pointing to the config file
				configDir := filepath.Join(tmpDir, "config")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatalf("Failed to create config directory: %v", err)
				}
				symlinkPath := filepath.Join(configDir, "config.yaml")
				if err := os.Symlink(configFile, symlinkPath); err != nil {
					t.Fatalf("Failed to create symlink: %v", err)
				}

				return symlinkPath
			},
			expectError:   false,
			errorContains: "",
		},
		{
			name: "directory instead of file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configDir := filepath.Join(tmpDir, "config.yaml")
				if err := os.MkdirAll(configDir, 0755); err != nil {
					t.Fatalf("Failed to create config directory: %v", err)
				}
				return configDir
			},
			expectError:   true,
			errorContains: "not a regular file",
		},
		{
			name: "file too large",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "config.yaml")

				// Create a file larger than 1MB
				largeContent := make([]byte, 1024*1024+1)
				if err := os.WriteFile(configFile, largeContent, 0644); err != nil {
					t.Fatalf("Failed to write large config file: %v", err)
				}

				return configFile
			},
			expectError:   true,
			errorContains: "too large",
		},
		{
			name: "valid config file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				configFile := filepath.Join(tmpDir, "config.yaml")
				if err := os.WriteFile(configFile, []byte("project:\n  name: TestApp\n"), 0644); err != nil {
					t.Fatalf("Failed to write config file: %v", err)
				}
				return configFile
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath := tt.setupFunc(t)
			_, err := LoadConfig(configPath)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q but got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}
