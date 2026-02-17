# MacReleaser Phase 7: Pipeline Robustness — Implementation Plan

## Overview

Phase 7 addresses four gaps discovered during integration testing with TestApp. These range from UX issues (eager env var resolution breaking `snapshot`/`build` commands) to misleading config fields (architectures). The fixes improve the pipeline's robustness for real-world usage without changing its fundamental architecture.

## Scope

### In Scope
- Tolerant environment variable substitution (missing vars left as literals, validated in CheckPipes)
- `env.CheckResolved()` function for field-level env var validation
- Skip guards in CheckPipes for release, homebrew, and notarize
- `--skip-notarize` flag on `build` and `snapshot` commands
- `SkipNotarize` context field (following `SkipPublish` pattern)
- Hardened runtime disabled when notarization is skipped
- Output directory conflict detection (fail early if `dist/<name>/<version>/` exists)
- Removal of unused `build.architectures` config field

### Out of Scope
- `--skip-notarize` on `release` (published builds must be notarized)
- Config migration tooling for removed fields
- Additional skip flags beyond notarize

## Technical Decisions

### Tolerant Env Substitution

The `replaceEnvVarsInString()` function in `pkg/env/env.go` now leaves unresolved `env(VAR)` references as literals instead of returning an error. This moves validation responsibility to the CheckPipes, which can skip validation for pipes that won't run.

A new `CheckResolved(value, field string) error` function detects leftover `env(...)` patterns and produces clear errors like `"notarize.password: environment variable APPLE_PASSWORD is not set"`.

### Skip Guards in CheckPipes

CheckPipes for skippable pipes return `skipError` when their corresponding skip flag is set:
- `release.CheckPipe` and `homebrew.CheckPipe`: skip when `ctx.SkipPublish`
- `notarize.CheckPipe`: skip when `ctx.SkipNotarize`

This prevents validation errors for config fields that reference unset env vars in pipes that won't run.

### SkipNotarize Design

Notarization is a required build step for distributable macOS apps — an un-notarized Developer ID app can't be run by end users. The `--skip-notarize` flag is for quick local pipeline validation only, available on `build` and `snapshot` but not `release`.

When `SkipNotarize` is set:
1. `notarize.CheckPipe` returns `skipError` (no credential validation)
2. `notarize.Pipe` returns `skipError` (no Apple submission)
3. `sign.Pipe` disables hardened runtime (not needed without notarization)

### Output Directory Conflict

The build pipe checks `os.Stat(outputDir)` before `os.MkdirAll`. If the directory exists, it returns `"output directory <path> already exists — remove it or use a different version"`. This prevents the confusing "unsealed contents present in the bundle root" codesign error.

### Architectures Removal

The `build.architectures` config field was validated but never passed to `xcodebuild`. The Xcode project controls architectures. Since config parsing uses `yaml.Strict()`, existing configs with `architectures:` will fail with a clear unknown-field error.

## Files Modified

| File | Change |
|------|--------|
| `pkg/env/env.go` | Tolerant substitution + `CheckResolved()` |
| `pkg/env/env_test.go` | Updated tests for tolerant behavior + `CheckResolved` tests |
| `pkg/context/context.go` | Added `SkipNotarize` field |
| `pkg/config/config.go` | Removed `Architectures` from `BuildConfig` |
| `pkg/config/example.go` | Removed `Architectures` from `ExampleConfig()` |
| `pkg/config/config_test.go` | Removed architectures refs, added tolerant env test |
| `pkg/cli/shared.go` | Added `withSkipNotarize()` option |
| `pkg/cli/root.go` | Added `--skip-notarize` flag to build and snapshot |
| `pkg/cli/build.go` | Read `--skip-notarize` flag |
| `pkg/cli/snapshot.go` | Read `--skip-notarize` flag |
| `internal/pipe/build/check.go` | Removed architecture validation, added `env.CheckResolved` |
| `internal/pipe/build/check_test.go` | Removed architecture test cases |
| `internal/pipe/build/pipe.go` | Added output directory existence check |
| `internal/pipe/build/pipe_test.go` | Added test for pre-existing output directory |
| `internal/pipe/sign/check.go` | Added `env.CheckResolved` |
| `internal/pipe/sign/pipe.go` | `hardenedRuntime` conditioned on `!ctx.SkipNotarize` |
| `internal/pipe/notarize/check.go` | Added skip guard + `env.CheckResolved` |
| `internal/pipe/notarize/pipe.go` | Added `skipError` type + skip guard |
| `internal/pipe/notarize/check_test.go` | Added skip test |
| `internal/pipe/notarize/pipe_test.go` | Added skip test |
| `internal/pipe/release/check.go` | Added skip guard + `env.CheckResolved` |
| `internal/pipe/homebrew/check.go` | Added skip guard + `env.CheckResolved` |
| `internal/pipe/project/check.go` | Added `env.CheckResolved` |
| `testapp/.macreleaser.yaml` | Removed `architectures` |

## Verification

1. `go test ./...` — all tests pass
2. Config without `architectures:` parses; config WITH `architectures:` produces strict-parsing error
3. Pre-existing output directory triggers early error
4. `macreleaser build --skip-notarize` skips notarization, no hardened runtime
5. `macreleaser build` with credentials runs notarization normally
6. `macreleaser release` has no `--skip-notarize` flag
7. Unset `HOMEBREW_TAP_TOKEN`, run `macreleaser build --skip-notarize` — succeeds
8. Run `macreleaser release` without token — clear error about missing variable
