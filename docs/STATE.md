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

**Phase 3-6** (To be planned): Code signing, notarization, releases, Homebrew integration

## Milestone 2: CI/CD

**To be planned**

## Milestone 3: Custom GitHub action

**To be planned**

## Milestone 4: Enhanced Features

**To be planned**
 
