# MacReleaser Plan

This document describes the implementation plan of MacReleaser. The current state of development is tracked in [STATE.md](STATE.md).

## Implementation Plan

### 🎯 Milestone 1: Steel Thread Release Automation

- Core Foundation + Minimal Config Schema
- Basic Build with One Arch, One Config
- Basic Code Signing
- Basic Notarization
- Basic Release to GitHub
- Basic Cask Generation and Custom Tap Support Only

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
