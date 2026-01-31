# Current State of MacReleaser Development

## Milestone 1: Steel Thread Release Automation

**Phase 1** (Complete): Core foundation, configuration parsing, validation
- ✅ CLI framework
- ✅ Configuration system with YAML parsing
- ✅ Environment variable substitution
- ✅ Validation system
- ✅ GitHub API client setup

**Phase 2** (Complete): Basic build, archive, and packaging pipeline
- ✅ Pipeline split into validation and execution stages (`ValidationPipes` / `ExecutionPipes`)
- ✅ Validation pipes renamed to `CheckPipe` structs within original packages
- ✅ `Context` expanded with `Version` and `Artifacts` fields
- ✅ Git version resolution (`pkg/git/`)
- ✅ Workspace/project auto-detection (`pkg/build/detect.go`)
- ✅ `xcodebuild archive` integration (`pkg/build/xcodebuild.go`)
- ✅ `.app` extraction from `.xcarchive`
- ✅ ZIP packaging via `ditto` (`pkg/archive/zip.go`)
- ✅ DMG packaging via `hdiutil` (`pkg/archive/dmg.go`)
- ✅ `build`, `release`, and `snapshot` commands wired to two-stage pipeline
- ✅ Shared CLI boilerplate extracted (`runPipelineCommand` in `pkg/cli/shared.go`)

**Phase 3** (Complete): Basic code signing
- ✅ `codesign` integration: sign `.app` bundle with identity from config
- ✅ Signature verification with `codesign --verify --deep --strict`
- ✅ Keychain identity validation via `security find-identity -v -p codesigning`
- ✅ Sign execution pipe registered between build and archive in `ExecutionPipes`
- ✅ Actionable error messages for common failures (identity not found, codesign missing, extended attributes)
- ✅ Unit tests for argument construction and identity validation (pure functions, no system deps)

**Phase 4-6** (To be planned): Notarization, releases, Homebrew integration

## Milestone 2: CI/CD

**To be planned**

## Milestone 3: Custom GitHub action

**To be planned**

## Milestone 4: Enhanced Features

**To be planned**
 
