# MacReleaser Plan

This document describes the implementation plan of MacReleaser. The current state of development is tracked in [STATE.md](STATE.md).

## Implementation Plan

### 🎯 Milestone 1: Steel Thread Release Automation

**Goal**: End-to-end workflow from init to GitHub release with bare minimum viable functionality

#### Phase 1: Core Foundation + Minimal Config Schema
- **📋 [Detailed Implementation Plan](PLAN-MILESTONE-01-PHASE-01.md)**
- Basic CLI structure and configuration parsing
- YAML validation and error handling
- GitHub API client setup
- **In Scope**: `project.name/scheme`, `build.configuration`, `sign.identity`, `notarize.*`, `release.github.*`, basic `archive.formats`
- **Out of Scope**: `project.workspace`, multiple architectures, custom archive options, Homebrew integration

#### Phase 2: Basic Build with One Arch, One Config
- **📋 [Detailed Implementation Plan](PLAN-MILESTONE-01-PHASE-02.md)**
- xcodebuild integration for single target
- Archive creation for current machine architecture only
- Basic DMG/ZIP packaging with defaults
- **In Scope**: Build `.app` and create `.dmg`/`.zip`
- **Out of Scope**: Parallel builds, custom archive styling

#### Phase 3: Basic Code Signing
- **📋 [Detailed Implementation Plan](PLAN-MILESTONE-01-PHASE-03.md)**
- Developer ID certificate handling from keychain
- codesign integration with identity from config
- Basic signature verification
- **In Scope**: Simple signing with `--force` flag support
- **Out of Scope**: Multiple certificates, custom signing options

#### Phase 4: Basic Notarization
- **📋 [Detailed Implementation Plan](PLAN-MILESTONE-01-PHASE-04.md)**
- notarytool integration with Apple ID auth
- Upload and basic polling workflow
- Ticket stapling to archive
- **In Scope**: Basic notarization with default timeouts
- **Out of Scope**: Custom retry logic, advanced notarization options

#### Phase 5: Basic Release to GitHub
- **📋 [Detailed Implementation Plan](PLAN-MILESTONE-01-PHASE-05.md)**
- GitHub releases API integration
- Asset upload (single archive)
- Basic version tagging from git
- **In Scope**: Release current archive to GitHub
- **Out of Scope**: Changelog generation, draft releases, multiple assets

#### Phase 6: Basic Cask Generation and Custom Tap Support Only
- **📋 [Detailed Implementation Plan](PLAN-MILESTONE-01-PHASE-06.md)**
- Cask generation from release info
- Custom tap direct commits (write permissions)
- Basic SHA256 hash calculation
- **In Scope**: Simple cask with version/URL/hash
- **Out of Scope**: Official tap PRs, cask customization, dependency detection

#### Phase 7: Pipeline Robustness
- **📋 [Detailed Implementation Plan](PLAN-MILESTONE-01-PHASE-07.md)**
- Tolerant env var resolution with deferred validation in CheckPipes
- `--skip-notarize` flag for quick local pipeline validation
- Output directory conflict detection
- Removed unused `architectures` config field
- **In Scope**: Skip guards, env.CheckResolved(), output dir checks
- **Out of Scope**: Additional skip flags, config migration tooling

### Milestone 2: CI/CD
- Use goreleaser to implement CI/CD for macreleaser itself
- GitHub actions workflows to build, test, and release macreleaser
- Basic integration testing with test Xcode project
- **Out of Scope**: Custom GitHub action

### Milestone 3: Custom GitHub action
- GitHub action that wraps macreleaser (`action-macreleaser`)
- Basic integration testing with test Xcode project

### Milestone 4: Enhanced Features
- Homebrew official tap integration
- Changelog generation from git history
- Multiple architecture builds
- Advanced archive customization (DMG styling, etc.)
- Sparkle update feed generation
