# MacReleaser Phase 2: Basic Build + Packaging Implementation Plan

## Overview

Phase 2 delivers the first real build pipeline: run a single-arch Xcode build, locate the resulting `.app`, and package it into `.zip` and/or `.dmg` with default settings. This phase intentionally keeps scope narrow: one scheme, one configuration, current machine architecture, and no signing/notarization.

## Scope

### In Scope
- Invoke `xcodebuild` to produce an unsigned archive
- Export or locate the `.app` from a single archive
- Package output as `.zip` and/or `.dmg` using defaults
- Integrate the build step into `macreleaser build` (and reuse in later phases)
- Auto-detect `.xcworkspace`/`.xcodeproj` when `project.workspace` is not set
- Derive version from git tags

### Out of Scope
- Multiple architectures or parallel builds
- Custom DMG styling or advanced packaging options
- Code signing, notarization, or release upload

## Technical Decisions

### Pipe Architecture Evolution

Phase 1 pipes are validation-only. Phase 2 introduces execution pipes that produce artifacts. To avoid naming conflicts:

- **Rename existing validation pipes** from `build.Pipe{}`, `archive.Pipe{}`, etc. to `buildcheck.Pipe{}`, `archivecheck.Pipe{}`, etc. under `internal/pipe/<name>check/`
- **Reclaim the original names** (`build.Pipe{}`, `archive.Pipe{}`) for the new execution pipes under `internal/pipe/build/` and `internal/pipe/archive/`
- **Split the pipe registry** into two ordered slices:
  - `ValidationPipes` — all renamed `*check` pipes (run by `check`, and as the first stage of `build`/`release`/`snapshot`)
  - `ExecutionPipes` — new execution pipes (run after validation succeeds)
- The `Piper` interface remains unchanged (`String()` + `Run()`)

### Runtime State: Artifacts Struct

Execution pipes populate runtime output state that subsequent pipes consume. This is kept separate from input config:

```go
// pkg/context/context.go

type Artifacts struct {
    BuildOutputDir string   // dist/<project>/<version>/
    ArchivePath    string   // path to .xcarchive
    AppPath        string   // path to extracted .app
    Packages       []string // paths to .zip, .dmg outputs
}

type Context struct {
    StdCtx    context.Context
    Config    *config.Config
    Logger    *logrus.Logger
    Version   string      // derived from git tag
    Artifacts *Artifacts  // populated by execution pipes
}
```

### CLI Command Flow

All commands that run execution pipes **must run validation first**:

- `macreleaser check` — runs `ValidationPipes` only
- `macreleaser build` — runs `ValidationPipes`, then `ExecutionPipes` (build + archive)
- `macreleaser release` / `snapshot` — same as `build`, plus future signing/notarization/upload pipes (not enabled in Phase 2)

If validation fails, execution never starts.

### Version Source

- Derive version from the latest git tag using `git describe --tags`
- Populate `ctx.Version` before any pipes run
- Fail with a clear error if no git tags exist (e.g., "no git tags found — tag your release with `git tag v1.0.0`")

### Workspace/Project Auto-Detection

When `project.workspace` is not set in config:

1. Look for a single `.xcworkspace` in the current directory
2. If none found, look for a single `.xcodeproj`
3. If neither or multiple are found, fail with a clear error listing what was found

This runs as part of the build execution pipe, before invoking `xcodebuild`.

### xcodebuild Argument Mapping

Config fields map to `xcodebuild archive` flags:

| Config Field | xcodebuild Flag |
|---|---|
| `project.scheme` | `-scheme <value>` |
| `project.workspace` (or auto-detected) | `-workspace <value>` (for `.xcworkspace`) or `-project <value>` (for `.xcodeproj`) |
| `build.configuration` | `-configuration <value>` |
| (derived) | `-archivePath dist/<project>/<version>/<scheme>.xcarchive` |

### Build Output Strategy
- Use `xcodebuild archive` to produce a `.xcarchive`
- Use derived paths under `dist/` for artifacts

### Packaging Defaults
- **ZIP**: Use `ditto -c -k --sequesterRsrc --keepParent` on the `.app`
- **DMG**: Use `hdiutil create` with a default volume name and size
- Only include the single `.app` in the archive

### Paths and Naming
- `dist/` remains the default output root
- Archive path: `dist/<project>/<version>/<scheme>.xcarchive`
- App path: `dist/<project>/<version>/<AppName>.app`
- Package names: `dist/<project>/<version>/<AppName>-<version>.{zip,dmg}`

## Detailed Implementation Tasks

### Task 2.0: Rename Validation Pipes
**Subtasks:**
2.0.1 Rename existing validation pipes:
- `internal/pipe/build/` → `internal/pipe/buildcheck/` (struct becomes `buildcheck.Pipe{}`)
- `internal/pipe/archive/` → `internal/pipe/archivecheck/` (struct becomes `archivecheck.Pipe{}`)
- Rename all other Phase 1 validation pipes similarly: `projectcheck`, `signcheck`, `notarizecheck`, `releasecheck`, `homebrewcheck`

2.0.2 Split `pkg/pipe/registry.go`:
- Replace `All` with two slices: `ValidationPipes` and `ExecutionPipes`
- `ValidationPipes` contains all renamed `*check` pipes
- `ExecutionPipes` is initially empty, populated in later tasks

2.0.3 Update `pkg/pipeline/pipeline.go`:
- Add ability to run validation pipes, then execution pipes, as separate stages
- `check` command runs `ValidationPipes` only
- `build`/`release`/`snapshot` run both stages

2.0.4 Update all existing tests referencing old pipe names

### Task 2.1: Version Resolution
**Subtasks:**
2.1.1 Create `pkg/git/version.go`:
- Run `git describe --tags` to extract the latest tag
- Parse and validate the tag as a version string
- Return clear error if no tags exist

2.1.2 Populate `ctx.Version` during context initialization, before pipes run

### Task 2.2: Artifacts Struct + Context Updates
**Subtasks:**
2.2.1 Add `Artifacts` struct to `pkg/context/context.go`:
- `BuildOutputDir`, `ArchivePath`, `AppPath`, `Packages []string`

2.2.2 Add `Version string` field to `Context`

2.2.3 Initialize `Artifacts` as empty struct when context is created

### Task 2.3: Workspace Auto-Detection
**Subtasks:**
2.3.1 Create `pkg/build/detect.go`:
- Implement workspace/project auto-detection logic
- Look for single `.xcworkspace`, fall back to `.xcodeproj`
- Return detected path and type (workspace vs project)
- Fail with clear error listing candidates if ambiguous

### Task 2.4: Build Execution Pipe
**Subtasks:**
2.4.1 Create `internal/pipe/build/pipe.go` (execution pipe):
- Run workspace auto-detection if `project.workspace` is empty
- Build `xcodebuild archive` arguments from config (see argument mapping table)
- Execute `xcodebuild archive`
- Capture stdout/stderr for logging
- Set `ctx.Artifacts.ArchivePath` on success

2.4.2 Create `pkg/build/xcodebuild.go`:
- Minimal command runner wrapper
- Builds argument list from config fields
- Provides actionable error messages (missing scheme, xcodebuild not found, archive failed)

2.4.3 Register `build.Pipe{}` in `ExecutionPipes` registry

### Task 2.5: Archive App Extraction
**Subtasks:**
2.5.1 Locate `.app` in the `.xcarchive`:
- Expect `.xcarchive/Products/Applications/<AppName>.app`
- Copy the `.app` to `dist/<project>/<version>/`
- Set `ctx.Artifacts.AppPath`

2.5.2 Validate `.app` existence and size > 0

### Task 2.6: Packaging (ZIP + DMG)
**Subtasks:**
2.6.1 ZIP packager:
- Create `pkg/archive/zip.go`
- Use `ditto` to preserve macOS metadata
- Append output path to `ctx.Artifacts.Packages`

2.6.2 DMG packager:
- Create `pkg/archive/dmg.go`
- Use `hdiutil create` with:
  - Volume name: `<AppName> <version>`
  - FS: HFS+
- Append output path to `ctx.Artifacts.Packages`

2.6.3 Archive execution pipe:
- Create `internal/pipe/archive/pipe.go` (execution pipe)
- Reads `archive.formats` from config and packages accordingly
- Register in `ExecutionPipes` after build pipe

### Task 2.7: CLI Integration
**Subtasks:**
2.7.1 Wire `macreleaser build` to run `ValidationPipes` then `ExecutionPipes`
2.7.2 Wire `macreleaser release` and `snapshot` to use the same two-stage pipeline (but no signing/notarization pipes registered yet)
2.7.3 `macreleaser check` continues to run `ValidationPipes` only

### Task 2.8: Logging and Output Summary
**Subtasks:**
2.8.1 Standard log format for build steps (pipe name, action, duration)
2.8.2 Print a concise summary of produced artifacts from `ctx.Artifacts.Packages`

### Task 2.9: Tests
**Subtasks:**
2.9.1 Unit tests for `xcodebuild` argument construction (mock command runner, no real invocation)
2.9.2 Unit tests for workspace auto-detection (mock filesystem)
2.9.3 Unit tests for archive path building and file operations
2.9.4 Unit tests for git version resolution (mock git output)
2.9.5 Integration test with fake `.xcarchive` fixture (app extraction + packaging)
2.9.6 Tests for the renamed validation pipes still pass
2.9.7 All tests assume macOS — no Linux portability concerns

## Acceptance Criteria

### Build
- [ ] `macreleaser build` runs `xcodebuild archive` for a single scheme
- [ ] `.xcarchive` is created in `dist/` with deterministic naming
- [ ] `.app` is copied to `dist/` and validated
- [ ] Version is derived from git tags
- [ ] Workspace/project is auto-detected when not configured

### Packaging
- [ ] ZIP packaging creates a valid archive using `ditto`
- [ ] DMG packaging creates a mountable image using `hdiutil`
- [ ] Output paths are logged and included in final summary

### CLI & Pipeline
- [ ] Validation pipes always run before execution pipes
- [ ] `macreleaser check` runs only validation (no execution)
- [ ] `macreleaser build` runs validation then build+archive execution
- [ ] `macreleaser release` and `snapshot` reuse the same two-stage pipeline

### Architecture
- [ ] Existing validation pipes renamed to `*check` pattern
- [ ] Registry split into `ValidationPipes` and `ExecutionPipes`
- [ ] Runtime state stored in `ctx.Artifacts`, separate from config
- [ ] All existing tests updated and passing after rename

### Quality
- [ ] New functionality has unit tests
- [ ] Error messages are actionable and reference config fields
- [ ] Auto-detection errors list what was found for easy debugging

## Risks and Mitigations

- **Risk**: `xcodebuild` failures due to missing Xcode CLI tools
  - **Mitigation**: Preflight check and a clear error message with install hint
- **Risk**: `.app` path assumptions break with nonstandard Xcode setups
  - **Mitigation**: Validate `Products/Applications` path and expose error details
- **Risk**: DMG creation fails on older macOS
  - **Mitigation**: Default to HFS+ and allow future options in later phases
- **Risk**: `git describe --tags` fails in repos with no tags
  - **Mitigation**: Clear error message suggesting `git tag v1.0.0`
- **Risk**: Auto-detection finds multiple workspaces/projects
  - **Mitigation**: List all candidates in the error message so the user can set `project.workspace` explicitly

## Dependencies and Tools

- `xcodebuild` (Xcode CLI tools)
- `ditto` (ZIP packaging)
- `hdiutil` (DMG creation)
- `git` (version resolution)
- Go standard library for file operations and process execution

## Notes for Future Phases

- Phase 3 will attach `codesign` to the `.app` and possibly repackage
- Phase 4 will notarize the DMG/ZIP and staple tickets
- Phase 5 will upload the packaged artifacts to GitHub
