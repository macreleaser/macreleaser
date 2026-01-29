# MacReleaser: macOS App Release Automation Tool

## Overview

MacReleaser is a specialized release automation tool focused exclusively on releasing Developer ID signed macOS applications built using Xcode. Inspired by goreleaser, but purpose-built for the macOS app ecosystem.

## Core Vision

A configuration-driven tool that automates the complete workflow:
**Build → Archive → Sign → Notarize → Release → Homebrew Distribution**

## Key Features

### 1. Build Automation
- Integration with `xcodebuild` for building archives
- Support for Xcode projects, workspaces, and Swift packages
- Multiple scheme and configuration support
- Parallel builds for different architectures (arm64, x86_64)

### 2. Code Signing & Notarization
- Developer ID certificate handling
- Automated code signing using `codesign`
- Apple notarization using `notarytool` (replacing `altool`)
- Stapling notarization tickets
- Verification with `spctl`

### 3. Distribution Formats
- `.app` bundles
- `.dmg` disk images with custom styling
- `.zip` archives
- Sparkle update packages

### 4. Release Management
- GitHub releases with automatic asset uploads
- Version tagging from git
- Changelog generation from git history
- Release notes integration

### 5. Homebrew Integration
- **Custom Tap Support**: Direct commits to personal taps (with write permissions)
- **Official Tap Support**: Pull requests to `homebrew/cask`
- Automatic cask generation with proper SHA256 hashes
- Smart dependency detection from app bundles

## Technical Architecture

### Language: Go
- Single binary distribution (no runtime dependencies)
- Excellent CLI tool ecosystem
- Good cross-platform support (for CI/CD)
- Proven track record from goreleaser

### Core Components
1. **Config Parser** - YAML configuration handling
2. **Builder** - xcodebuild integration
3. **Signer** - codesign and certificate management  
4. **Notary** - notarytool integration
5. **Packager** - DMG/ZIP creation
6. **Releaser** - GitHub integration
7. **Homebrew** - Cask generation and tap management

## Configuration

### File: `.macreleaser.yaml`

```yaml
# Project configuration
project:
  name: MyApp
  scheme: MyApp
  workspace: MyApp.xcworkspace
  
# Build settings
build:
  configuration: Release
  architectures: [arm64, x86_64]
  
# Code signing
sign:
  identity: "Developer ID Application: Your Name (TEAM_ID)"
  
# Notarization
notarize:
  apple_id: env(APPLE_ID)
  team_id: env(TEAM_ID)
  password: env(APPLE_APP_SPECIFIC_PASSWORD)
  
# Archive formats
archive:
  formats: [dmg, zip]
  dmg:
    background: background.png
    icon_size: 128
    
# GitHub release
release:
  github:
    owner: yourname
    repo: myapp
    draft: false
    
# Homebrew distribution
homebrew:
  # Custom tap (direct commits)
  tap:
    owner: yourname
    name: homebrew-tap
    token: env(HOMEBREW_TAP_TOKEN)
    
  # Official homebrew/cask (pull requests)  
  official:
    enabled: true
    token: env(HOMEBREW_OFFICIAL_TOKEN)
    auto_merge: false
    assignees: ["your-github-username"]
    
  # Cask metadata
  cask:
    name: myapp
    desc: "My awesome macOS application"
    homepage: "https://github.com/yourname/myapp"
    license: MIT
```

## CLI Interface

```bash
macreleaser init          # Generate config
macreleaser check         # Validate configuration  
macreleaser build         # Build and archive only
macreleaser release       # Full release process including Homebrew
macreleaser snapshot      # Test release with snapshot version
```

## Homebrew Integration Strategy

### Custom Tap Integration (Direct Commits)
- **Repository**: `username/homebrew-tap` 
- **Directory**: `Casks/`
- **Method**: Direct commits via GitHub API
- **Authentication**: GitHub token with write permissions

### Official Homebrew Cask Tap (Pull Requests)
- **Repository**: `Homebrew/homebrew-cask`
- **Method**: Create pull requests using GitHub API
- **Process**: PR creation following Homebrew contribution guidelines
- **Authentication**: GitHub token with PR permissions

### Generated Cask Template

```ruby
cask "myapp" do
  version "1.2.3"
  sha256 "generated_sha256_hash"
  
  url "https://github.com/yourname/myapp/releases/download/v#{version}/MyApp-#{version}.dmg"
  name "MyApp"
  desc "My awesome macOS application"
  homepage "https://github.com/yourname/myapp"
  
  app "MyApp.app"
  
  zap trash: [
    "~/Library/Application Support/MyApp",
    "~/Library/Preferences/com.yourname.myapp.plist"
  ]
end
```

## GitHub Action Integration

A companion GitHub Action `action-macreleaser` that:
- Sets up macOS runner
- Installs Xcode command line tools
- Handles certificate and key provisioning
- Runs macreleaser with proper environment variables
- Manages authentication tokens securely

## Dependencies

### External Tools (Expected)
- Xcode and Xcode Command Line Tools
- `xcodebuild` - Build and archive
- `codesign` - Code signing
- `notarytool` - Notarization
- `spctl` - Security assessment
- `xcrun` - Xcode toolchain utilities

### Runtime Dependencies
- None (single Go binary)
- GitHub tokens for API access
- Apple credentials for notarization

## Security Considerations

- No storage of sensitive data in configuration
- Environment variables for all secrets
- Certificate and key handling via macOS keychain
- Token-based authentication for external services
- Verification of code signatures and notarization

## Testing Strategy

- Unit tests for all core components
- Integration tests with sample Xcode projects
- End-to-end tests with mock GitHub/Apple services
- CI testing on macOS runners
- Snapshot release testing

## Advantages over General Tools

- **macOS-Native**: Built specifically for macOS app distribution
- **Xcode Integration**: Native support for Xcode build system
- **App Bundle Awareness**: Understands macOS app structure
- **Homebrew Compliance**: Follows official cask contribution guidelines
- **Security Focused**: Proper handling of Developer ID signing and notarization
- **Simplified Configuration**: Opinionated defaults for common macOS app patterns

## Comparison with goreleaser

| Feature | goreleaser | macreleaser |
|---------|------------|-------------|
| Primary Focus | Go binaries | macOS apps |
| Build System | Go compiler | xcodebuild |
| Code Signing | Basic | Developer ID + Notarization |
| Distribution | Multiple platforms | macOS-only |
| Package Managers | Homebrew formulas | Homebrew casks |
| Archive Formats | tar.gz, zip | DMG, zip, app bundles |
| Platform Dependencies | Minimal | Xcode tools required |

## Future Enhancements

- Sparkle update feed generation
- Multiple certificate/profile support
- Custom notarization workflows
- Support for iOS/iPadOS apps
- Integration with other package managers
- Advanced DMG customization options

## Contributing Guidelines

- Follow Go conventions and best practices
- Comprehensive testing for all features
- Documentation updates with new features
- Security review for any certificate/key handling
- Homebrew compliance verification

## License

MIT License (similar to goreleaser for consistency)
