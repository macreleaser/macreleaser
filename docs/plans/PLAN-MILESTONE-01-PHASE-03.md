# MacReleaser Phase 3: Basic Code Signing — Implementation Plan

## Overview

Phase 3 adds code signing to the release pipeline. A new sign execution pipe runs between build and archive so the `.app` bundle is signed in-place before being packaged into `.zip`/`.dmg`. This phase covers signing with a single identity from the macOS keychain using `codesign --deep --force`, followed by signature verification. No config schema or context changes are needed — `SignConfig.Identity` and `sign.CheckPipe` already exist from Phase 1.

## Scope

### In Scope
- `codesign` integration: sign the `.app` at `ctx.Artifacts.AppPath` using the identity from config
- Signature verification with `codesign --verify --deep --strict` after signing
- Keychain identity validation using `security find-identity -v -p codesigning`
- Sign execution pipe registered between build and archive in `ExecutionPipes`
- Actionable error messages for common failures (identity not found, codesign missing, verification failure)
- Unit tests for argument construction and identity validation (pure functions, no system deps)

### Out of Scope
- Multiple signing identities
- Custom signing options or flags beyond `--deep --force`
- Entitlements files
- Hardened Runtime (`--options runtime`) — deferred to Phase 4 since notarization requires it
- `spctl --assess` (Gatekeeper assessment) — deferred to Phase 4

## Technical Decisions

### Pipeline Order Change

Current `ExecutionPipes` in `pkg/pipe/registry.go`:
```go
var ExecutionPipes = []Piper{
    build.Pipe{},   // Build and archive with xcodebuild
    archive.Pipe{}, // Package into zip/dmg
}
```

New order:
```go
var ExecutionPipes = []Piper{
    build.Pipe{},   // Build and archive with xcodebuild
    sign.Pipe{},    // Code sign the .app bundle
    archive.Pipe{}, // Package into zip/dmg
}
```

Signing the `.app` in-place means the archive pipe automatically picks up the signed version without any changes.

### No Config or Context Changes

- `SignConfig.Identity` already exists (`pkg/config/config.go:39-41`)
- `sign.CheckPipe` already validates `sign.identity` is non-empty (`internal/pipe/sign/check.go`)
- Signing modifies the `.app` in-place at `ctx.Artifacts.AppPath` — no new `Artifacts` fields needed

### Identity Validation Strategy

`security find-identity -v -p codesigning` outputs lines like:
```
  1) AABBCCDD... "Developer ID Application: John Doe (TEAM123)"
  2) EEFF0011... "Apple Development: john@example.com (PERSONAL)"
     2 valid identities found
```

We parse this, check if the configured identity appears, and on failure list available identities so the user can correct their config.

## Detailed Implementation Tasks

### Task 3.0: Create `pkg/sign/codesign.go` — Codesign Command Wrapper

Following the pattern in `pkg/build/xcodebuild.go`.

**Subtasks:**

3.0.1. Define `CodesignArgs` struct with `Identity`, `AppPath`, `Deep`, `Force` fields

3.0.2. Implement `BuildCodesignArgs(args CodesignArgs) []string` — pure function returning flag list:
- `["--deep", "--force", "--sign", "<identity>", "<app_path>"]` (flags conditional on bool fields)

3.0.3. Implement `RunCodesign(args CodesignArgs) (string, error)`:
- `exec.LookPath("codesign")` check — if missing: `"codesign not found — install Xcode Command Line Tools with: xcode-select --install"`
- Build args via `BuildCodesignArgs`, execute with `CombinedOutput()`
- Parse common error patterns for actionable messages:
  - `"resource fork, Finder information, or similar detritus"` → suggest `xattr -cr`
  - General failure → wrap with full codesign output

### Task 3.1: Create `pkg/sign/verify.go` — Signature Verification

3.1.1. Define `VerifyArgs` struct with `AppPath`, `Deep`, `Strict` fields

3.1.2. Implement `BuildVerifyArgs(args VerifyArgs) []string` — pure function returning `["--verify", "--deep", "--strict", "<app_path>"]`

3.1.3. Implement `RunVerify(args VerifyArgs) (string, error)`:
- `exec.LookPath("codesign")` check
- Build args, execute, wrap failure with path and output

### Task 3.2: Create `pkg/sign/identity.go` — Keychain Identity Lookup

3.2.1. Implement `ParseIdentityOutput(output string) []string` — pure function that parses `security find-identity` output, extracting quoted identity strings from lines matching `N) <hash> "<identity>"`

3.2.2. Implement `ValidateIdentity(configuredIdentity string, availableIdentities []string) error` — pure function:
- Returns nil if `configuredIdentity` matches any available identity
- Returns error listing available identities with hint to run `security find-identity -v -p codesigning`

3.2.3. Implement `CheckIdentityInKeychain(configuredIdentity string) error` — convenience function:
- Runs `security find-identity -v -p codesigning`
- Parses output with `ParseIdentityOutput`
- Validates with `ValidateIdentity`

### Task 3.3: Create `internal/pipe/sign/pipe.go` — Sign Execution Pipe

3.3.1. Define `Pipe` struct with `String()` returning `"signing application"`

3.3.2. Implement `Run(ctx *context.Context) error`:
1. Guard: `ctx.Artifacts.AppPath == ""` → error `"no .app found to sign — ensure the build step completed successfully"`
2. Validate identity in keychain via `sign.CheckIdentityInKeychain()`
3. Run `sign.RunCodesign()` with `Deep: true, Force: true`
4. Verify with `sign.RunVerify()` with `Deep: true, Strict: true`
5. Log success

### Task 3.4: Register Sign Pipe in `pkg/pipe/registry.go`

Add `sign.Pipe{}` between `build.Pipe{}` and `archive.Pipe{}` in `ExecutionPipes`. No new import needed (already imported for `sign.CheckPipe{}`).

### Task 3.5: Tests for `pkg/sign/codesign_test.go`

Table-driven tests for `BuildCodesignArgs`:
- All flags enabled, deep only, force only, no flags, empty args

Table-driven tests for `BuildVerifyArgs`:
- Deep + strict, verify only

### Task 3.6: Tests for `pkg/sign/identity_test.go`

Table-driven tests for `ParseIdentityOutput`:
- Real-format sample output, empty output, malformed lines

Table-driven tests for `ValidateIdentity`:
- Identity found, not found (lists alternatives), empty list

### Task 3.7: Tests for `internal/pipe/sign/pipe_test.go`

- `TestPipeString` — verify `"signing application"`
- `TestPipeNoApp` — empty `AppPath` returns error containing `"no .app found to sign"`

### Task 3.8: Update `docs/STATE.md`

Update Phase 3 status to complete.

## Files Summary

### New Files (7)
| File | Purpose |
|------|---------|
| `pkg/sign/codesign.go` | `CodesignArgs`, `BuildCodesignArgs()`, `RunCodesign()` |
| `pkg/sign/verify.go` | `VerifyArgs`, `BuildVerifyArgs()`, `RunVerify()` |
| `pkg/sign/identity.go` | `ParseIdentityOutput()`, `ValidateIdentity()`, `CheckIdentityInKeychain()` |
| `internal/pipe/sign/pipe.go` | Sign execution pipe |
| `pkg/sign/codesign_test.go` | Tests for codesign/verify arg construction |
| `pkg/sign/identity_test.go` | Tests for identity parsing and validation |
| `internal/pipe/sign/pipe_test.go` | Tests for sign execution pipe |

### Modified Files (2)
| File | Change |
|------|--------|
| `pkg/pipe/registry.go` | Insert `sign.Pipe{}` between build and archive |
| `docs/STATE.md` | Update Phase 3 status |

### Unchanged Files
- `pkg/config/config.go` — `SignConfig.Identity` already exists
- `pkg/context/context.go` — no new `Artifacts` fields needed
- `internal/pipe/sign/check.go` — existing validation sufficient
- `internal/pipe/archive/pipe.go` — automatically picks up signed `.app`

## Verification

1. `go build ./...` — compiles without errors
2. `go test ./...` — all tests pass (new + existing)
3. `go vet ./...` — no issues
4. Manual test on a real Xcode project:
   - `macreleaser check` still runs validation only (no signing attempted)
   - `macreleaser build` runs build → sign → archive, producing signed `.zip`/`.dmg`
   - Verify the `.app` inside the archive is signed: `codesign --verify --deep --strict dist/<project>/<version>/<App>.app`

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| `codesign` not available | `exec.LookPath` check with `xcode-select --install` hint |
| Identity string mismatch | Pre-validate with `security find-identity`, list alternatives in error |
| Extended attributes cause codesign failure | Detect `"resource fork"` error pattern, suggest `xattr -cr` |
| Tests need real signing identity | Pure function tests have no system deps; pipe tests with real signing use `t.Skip()` |

## Notes for Future Phases

- **Phase 4** will add `--options runtime` (Hardened Runtime) to `CodesignArgs` for notarization
- **Phase 4** will add `spctl --assess` for Gatekeeper verification
- **Milestone 4** would replace `--deep` with per-framework signing and add entitlements
