package config

// ExampleConfig returns a configuration with example values for use with `macreleaser init`
func ExampleConfig() *Config {
	return &Config{
		Project: ProjectConfig{
			Name:   "MyApp",
			Scheme: "MyApp",
		},
		Build: BuildConfig{
			Configuration: "Release",
		},
		Sign: SignConfig{
			Identity: "Developer ID Application: Your Name (TEAM_ID)",
		},
		Notarize: NotarizeConfig{
			AppleID:  "env(APPLE_ID)",
			TeamID:   "env(TEAM_ID)",
			Password: "env(APPLE_APP_SPECIFIC_PASSWORD)",
		},
		Archive: ArchiveConfig{
			Formats: []string{"dmg", "zip"},
			DMG: DMGConfig{
				Background: "background.png",
				IconSize:   128,
			},
		},
		Release: ReleaseConfig{
			GitHub: GitHubConfig{
				Owner: "yourname",
				Repo:  "myapp",
				Draft: false,
			},
		},
		Homebrew: HomebrewConfig{
			Tap: TapConfig{
				Owner: "yourname",
				Name:  "homebrew-tap",
				Token: "env(HOMEBREW_TAP_TOKEN)",
			},
			Official: OfficialConfig{
				Enabled:   false,
				Token:     "env(HOMEBREW_OFFICIAL_TOKEN)",
				AutoMerge: false,
				Assignees: []string{"your-github-username"},
			},
			Cask: CaskConfig{
				Name:     "myapp",
				Desc:     "My awesome macOS application",
				Homepage: "https://github.com/yourname/myapp",
				License:  "MIT",
			},
		},
	}
}
