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

**Phase 4** (Complete): Basic notarization
- ✅ Hardened Runtime support (`--options runtime`) in `RunCodesign`, auto-enabled when notarization is configured
- ✅ `notarytool submit --wait` integration with Apple ID authentication (`pkg/notarize/notarytool.go`)
- ✅ Ticket stapling via `xcrun stapler staple` (`pkg/notarize/staple.go`)
- ✅ Gatekeeper verification via `spctl --assess --type execute` (`pkg/notarize/spctl.go`)
- ✅ Notarize execution pipe: temp ZIP → submit → staple → verify → cleanup (`internal/pipe/notarize/pipe.go`)
- ✅ Pipeline order: build → sign (Hardened Runtime) → notarize → archive
- ✅ Actionable error messages for auth failures, rejection, missing tickets, Gatekeeper rejection
- ✅ Unit tests for argument construction, submission ID parsing, and pipe guard conditions

**Phase 5** (Complete): Basic release to GitHub
- ✅ Release execution pipe: create GitHub release and upload archive assets (`internal/pipe/release/pipe.go`)
- ✅ `SkipPublish` context flag: `build` and `snapshot` skip publishing; only `release` creates a GitHub release
- ✅ `GitHubClient` on context for dependency injection and shared client across pipes
- ✅ Pipeline options: variadic `pipelineOption` pattern for `runPipelineCommand` (`pkg/cli/shared.go`)
- ✅ Content type detection for `.zip` and `.dmg` assets (`pkg/github/release.go`)
- ✅ Non-regular file filtering: `.app` directories in Packages skipped with warning
- ✅ Actionable error messages: missing token, release already exists, upload failure
- ✅ Release URL in artifact summary and `ctx.Artifacts.ReleaseURL` for downstream pipes
- ✅ Pipeline order: build → sign → notarize → archive → release
- ✅ Unit tests for content type helper, pipe guards, skip behavior, and mock client interactions

**Phase 6** (Complete): Homebrew cask generation and custom tap support
- ✅ SHA256 hash computation for archive packages (`pkg/homebrew/sha256.go`)
- ✅ Cask template rendering via Go `text/template` (`pkg/homebrew/cask.go`)
- ✅ Package selection: prefer `.zip`, fall back to `.dmg`
- ✅ Asset download URL construction from release config, version, and filename
- ✅ Version `v` prefix stripped for cask version stanza
- ✅ GitHub Contents API: `GetFileContents`, `CreateFile`, `UpdateFile` on `ClientInterface`
- ✅ Idempotent tap commits: create new cask or update existing (uses SHA for conflict detection)
- ✅ Separate `HomebrewClient` on context for tap authentication (may differ from `GITHUB_TOKEN`)
- ✅ Local cask file always generated in build output directory
- ✅ Homebrew execution pipe: hash → render → write local → commit to tap (`internal/pipe/homebrew/pipe.go`)
- ✅ Respects `SkipPublish` flag: `build`/`snapshot` skip entirely
- ✅ Cask path printed in artifact summary
- ✅ Pipeline order: build → sign → notarize → archive → release → homebrew
- ✅ Unit tests for SHA256, cask rendering, package selection, URL building, and pipe behavior

**Phase 7** (Complete): Pipeline robustness
- ✅ Removed unused `build.architectures` config field (validated but never passed to xcodebuild)
- ✅ Output directory conflict detection: fail early if `dist/<name>/<version>/` already exists
- ✅ `--skip-notarize` flag on `build` and `snapshot` commands for quick local pipeline validation
- ✅ `SkipNotarize` context field: skips notarize check/pipe, disables hardened runtime in sign pipe
- ✅ Tolerant env var substitution: missing `env(VAR)` references left as literals instead of failing
- ✅ `env.CheckResolved()` function for field-level validation of unresolved env vars in CheckPipes
- ✅ Skip guards in CheckPipes: release and homebrew skip when `SkipPublish`, notarize skips when `SkipNotarize`
- ✅ Enables `macreleaser build --skip-notarize` without Apple credentials or `HOMEBREW_TAP_TOKEN`

## Milestone 2: CI/CD

**Phase 1** (Complete): GoReleaser configuration
- ✅ Version package `const` → `var` for `-ldflags -X` injection
- ✅ `.goreleaser.yaml` with darwin/amd64 + darwin/arm64, changelog grouping
- ✅ Makefile `build` target injects version/commit/date via ldflags
- ✅ Verified with `goreleaser check` and `goreleaser build --snapshot --clean`

**Phase 2** (Complete): CI workflow
- ✅ `.github/workflows/ci.yml` with lint, test, and smoke-test jobs
- ✅ `.golangci.yml` with errcheck, govet, ineffassign, staticcheck, unused
- ✅ Lint and test on ubuntu-latest; smoke-test on macos-latest
- ✅ Existing lint issues fixed (unchecked error returns, proper noun nolint directives)

**Phase 3** (Planned): Release workflow
- Tag-triggered GoReleaser release
- Homebrew tap formula auto-publish

## Milestone 3: Custom GitHub action

**To be planned**

## Milestone 4: Enhanced Features

**To be planned**
 
