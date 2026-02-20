# MacReleaser Phase 4: Basic Notarization — Implementation Plan

## Overview

Phase 4 adds Apple notarization to the release pipeline. Notarization is required for macOS apps distributed outside the App Store — Apple scans the app for malicious content and issues a ticket that Gatekeeper verifies at launch. This phase integrates `notarytool` for submission, polls for approval using the built-in `--wait` flag, staples the ticket to the `.app`, and verifies the result with `spctl --assess`. The sign pipe is also updated to enable Hardened Runtime (`--options runtime`), which Apple requires before accepting a notarization submission.

## Scope

### In Scope
- Update `codesign` to support `--options runtime` (Hardened Runtime) when notarization is configured
- `xcrun notarytool submit` integration with Apple ID authentication and `--wait` polling
- Ticket stapling to the `.app` via `xcrun stapler staple`
- Gatekeeper verification via `spctl --assess --type execute`
- Notarize execution pipe inserted between sign and archive in `ExecutionPipes`
- Actionable error messages for common failures (invalid credentials, rejection, timeout)
- Unit tests for argument construction and output parsing (pure functions, no system deps)

### Out of Scope
- Custom retry logic or configurable retry counts
- Advanced notarization options (e.g., custom webhook, `--keychain-profile`)
- Notarization of individual frameworks or nested bundles
- Configurable timeout (uses `notarytool --wait` default behavior)
- Entitlements files (deferred to Milestone 4)

## Technical Decisions

### Pipeline Order Change

Notarization must happen after signing but before final archive creation. The notarize pipe creates a temporary ZIP for submission, waits for approval, staples the `.app`, and cleans up. The archive pipe then packages the stapled `.app` into the final `.zip`/`.dmg`, so end users receive notarized archives without any extra steps.

Current `ExecutionPipes`:
```go
var ExecutionPipes = []Piper{
    build.Pipe{},   // Build and extract .app
    sign.Pipe{},    // Code sign the .app bundle
    archive.Pipe{}, // Package into zip/dmg
}
```

New order:
```go
var ExecutionPipes = []Piper{
    build.Pipe{},      // Build and extract .app
    sign.Pipe{},       // Code sign with Hardened Runtime
    notarize.Pipe{},   // Submit, wait, staple .app
    archive.Pipe{},    // Package stapled .app into zip/dmg
}
```

### Why Notarize Before Archive

There are two viable strategies for notarization:

1. **Notarize before archive** (chosen): Submit a temp ZIP of the `.app`, staple the `.app`, then create final archives from the stapled `.app`.
2. **Notarize after archive**: Create archives first, submit the `.dmg` or `.zip`, then staple individual archives.

Strategy 1 is simpler because:
- Only one submission to Apple (the `.app` via temp ZIP), not one per archive format
- Stapling the `.app` means all downstream archives automatically contain the notarized app
- No need to re-create or re-staple archives
- `.zip` files cannot be stapled — only `.app`, `.dmg`, and `.pkg` can — so strategy 2 would leave ZIP archives without a stapled ticket anyway

### Hardened Runtime

Apple requires Hardened Runtime (`--options runtime`) for all notarized apps. Rather than adding a separate flag to the config, this is enabled automatically when `notarize.apple_id` is configured. This keeps config minimal and prevents the common mistake of forgetting to enable it.

The change is in `pkg/sign/codesign.go` — adding an `options` parameter to `RunCodesign`. The sign execution pipe reads `ctx.Config.Notarize.AppleID` to decide whether to pass it.

### notarytool Authentication

`notarytool` supports three authentication methods: Apple ID, API key, and keychain profile. Phase 4 uses Apple ID authentication only, which requires three fields already present in `NotarizeConfig`:

- `apple_id` — Apple ID email
- `team_id` — Developer Team ID
- `password` — App-specific password (should use `env(MACRELEASER_NOTARIZE_PASSWORD)`)

### Polling Strategy

`xcrun notarytool submit --wait` handles polling internally. It blocks until notarization completes or fails, printing progress to stderr. This is the simplest approach — no custom polling loop, no configurable intervals. Notarization typically completes in 1–15 minutes. The `--wait` flag has a built-in timeout (~60 minutes) which is sufficient for Phase 4.

### Gatekeeper Assessment

After stapling, `spctl --assess --type execute --verbose` verifies the app passes Gatekeeper. This was deferred from Phase 3 and is included here because it only makes sense after notarization — unsigned or un-notarized apps will always fail `spctl --assess` on modern macOS.

### No Config Changes

`NotarizeConfig` already has all required fields (`AppleID`, `TeamID`, `Password`). The existing `notarize.CheckPipe` already validates these are non-empty. No new config fields are needed for Phase 4.

### No Context Changes

The notarize pipe operates on `ctx.Artifacts.AppPath` (reads the signed `.app`) and `ctx.Artifacts.BuildOutputDir` (for temp ZIP placement). No new `Artifacts` fields are needed — the `.app` is stapled in-place, so the same path remains valid for the archive pipe.

## Detailed Implementation Tasks

### Task 4.0: Update `pkg/sign/codesign.go` — Add Hardened Runtime Support

**Subtasks:**

4.0.1. Change `RunCodesign` signature to accept a `hardenedRuntime bool` parameter:
```go
func RunCodesign(identity, appPath string, hardenedRuntime bool) (string, error)
```

4.0.2. When `hardenedRuntime` is true, include `--options`, `runtime` in the `codesign` argument list (before the identity flag):
```
codesign --deep --force --options runtime --sign <identity> <app_path>
```

4.0.3. Update all existing callers of `RunCodesign` — currently only `internal/pipe/sign/pipe.go`.

### Task 4.1: Update `internal/pipe/sign/pipe.go` — Pass Hardened Runtime Flag

**Subtasks:**

4.1.1. Determine whether Hardened Runtime should be enabled: check `ctx.Config.Notarize.AppleID != ""` as a proxy for "notarization is configured".

4.1.2. Pass the result to `sign.RunCodesign()`:
```go
hardenedRuntime := ctx.Config.Notarize.AppleID != ""
if hardenedRuntime {
    ctx.Logger.Info("Hardened Runtime enabled (required for notarization)")
}
output, err := sign.RunCodesign(identity, ctx.Artifacts.AppPath, hardenedRuntime)
```

### Task 4.2: Create `pkg/notarize/notarytool.go` — notarytool Command Wrapper

Following the pattern in `pkg/sign/codesign.go` and `pkg/build/xcodebuild.go`.

**Subtasks:**

4.2.1. Implement `BuildSubmitArgs(zipPath, appleID, teamID, password string) []string` — pure function returning the argument list:
```
["notarytool", "submit", "<zip_path>", "--apple-id", "<apple_id>",
 "--team-id", "<team_id>", "--password", "<password>", "--wait"]
```

4.2.2. Implement `RunSubmit(zipPath, appleID, teamID, password string) (string, error)`:
- `exec.LookPath("xcrun")` check — if missing: `"xcrun not found — install Xcode Command Line Tools with: xcode-select --install"`
- Build args via `BuildSubmitArgs`, execute `xcrun <args...>` with `CombinedOutput()`
- Parse common error patterns for actionable messages:
  - `"Unable to authenticate"` → `"notarytool authentication failed — verify apple_id, team_id, and password (use an app-specific password from appleid.apple.com)"`
  - `"Invalid"` or `"status: Invalid"` → `"Apple rejected the submission — run: xcrun notarytool log <submission-id> to view details"`
  - General failure → wrap with full notarytool output

4.2.3. Implement `ParseSubmissionID(output string) string` — pure function that extracts the submission UUID from notarytool output. The output contains a line like `  id: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx`. This is useful for error messages that reference the submission ID for log retrieval.

### Task 4.3: Create `pkg/notarize/staple.go` — Stapler Command Wrapper

**Subtasks:**

4.3.1. Implement `RunStaple(appPath string) (string, error)`:
- `exec.LookPath("xcrun")` check
- Execute `xcrun stapler staple <app_path>` with `CombinedOutput()`
- Parse common error patterns:
  - `"Could not find ticket"` → `"stapling failed — the notarization ticket was not found; ensure notarytool submission succeeded"`
  - General failure → wrap with full stapler output

### Task 4.4: Create `pkg/notarize/spctl.go` — Gatekeeper Assessment Wrapper

**Subtasks:**

4.4.1. Implement `RunAssess(appPath string) (string, error)`:
- `exec.LookPath("spctl")` check
- Execute `spctl --assess --type execute --verbose <app_path>` with `CombinedOutput()`
- Parse common error patterns:
  - `"rejected"` → `"Gatekeeper rejected the app — it may not be properly signed or notarized"`
  - General failure → wrap with full spctl output

### Task 4.5: Create `internal/pipe/notarize/pipe.go` — Notarize Execution Pipe

**Subtasks:**

4.5.1. Define `Pipe` struct with `String()` returning `"notarizing application"`

4.5.2. Implement `Run(ctx *context.Context) error`:

1. **Guard**: `ctx.Artifacts.AppPath == ""` → error `"no .app found to notarize — ensure the build and sign steps completed successfully"`

2. **Create temp ZIP** for submission:
   - Path: `<BuildOutputDir>/<AppName>-notarize.zip`
   - Use `pkg/archive.CreateZip()` (already exists) to ZIP the `.app`
   - Log: `"Creating temporary ZIP for notarization submission"`

3. **Submit to Apple**:
   - Call `notarize.RunSubmit(zipPath, appleID, teamID, password)`
   - Log: `"Submitting to Apple notary service (this may take several minutes)..."`
   - On error, include the submission ID if parseable for log retrieval

4. **Staple the `.app`**:
   - Call `notarize.RunStaple(ctx.Artifacts.AppPath)`
   - Log: `"Stapling notarization ticket"`

5. **Verify with Gatekeeper**:
   - Call `notarize.RunAssess(ctx.Artifacts.AppPath)`
   - Log: `"Verifying Gatekeeper assessment"`

6. **Clean up temp ZIP**:
   - `os.Remove(zipPath)` — log warning on failure but don't fail the pipe

7. **Log success**: `"Notarization complete: <app_path>"`

### Task 4.6: Register Notarize Pipe in `pkg/pipe/registry.go`

4.6.1. Add `notarize.Pipe{}` between `sign.Pipe{}` and `archive.Pipe{}` in `ExecutionPipes`.

4.6.2. Add the import for the notarize execution pipe (the import for `notarize` already exists for `notarize.CheckPipe{}`).

### Task 4.7: Tests for `pkg/notarize/notarytool_test.go`

Table-driven tests for `BuildSubmitArgs`:
- All fields populated — verify correct flag ordering and values
- Verify `--wait` is always included
- Verify password is passed as a positional argument to `--password` (not leaked in other flags)

Table-driven tests for `ParseSubmissionID`:
- Real notarytool output with UUID
- Output with no UUID
- Empty output

### Task 4.8: Tests for `pkg/notarize/staple_test.go`

- Test that `RunStaple` constructs the correct `xcrun stapler staple` command (verify via args, not execution)
- Since `RunStaple` calls `exec.Command` directly, these tests verify error wrapping patterns by checking error messages for expected substrings when commands fail

### Task 4.9: Tests for `pkg/notarize/spctl_test.go`

- Test that `RunAssess` constructs the correct `spctl --assess --type execute --verbose` command
- Verify error wrapping patterns for rejection and general failure cases

### Task 4.10: Tests for `internal/pipe/notarize/pipe_test.go`

Following the pattern in `internal/pipe/sign/pipe_test.go`:

- `TestPipeString` — verify `"notarizing application"`
- `TestPipeNoApp` — empty `AppPath` returns error containing `"no .app found to notarize"`

### Task 4.11: Update Sign Tests

4.11.1. Update `internal/pipe/sign/pipe_test.go`:
- Existing `TestPipeNoApp` still passes (signature unchanged from pipe's perspective)

4.11.2. No new test file for `pkg/sign/codesign_test.go` since there isn't one currently — the `RunCodesign` function calls `exec.Command` directly and is tested indirectly through the pipe tests. The Hardened Runtime flag adds `--options runtime` to the command, which is straightforward.

### Task 4.12: Update `docs/STATE.md`

Update Phase 4 status to complete with the list of deliverables.

## Files Summary

### New Files (6)
| File | Purpose |
|------|---------|
| `pkg/notarize/notarytool.go` | `BuildSubmitArgs()`, `RunSubmit()`, `ParseSubmissionID()` |
| `pkg/notarize/staple.go` | `RunStaple()` |
| `pkg/notarize/spctl.go` | `RunAssess()` |
| `internal/pipe/notarize/pipe.go` | Notarize execution pipe |
| `pkg/notarize/notarytool_test.go` | Tests for notarytool arg construction and output parsing |
| `internal/pipe/notarize/pipe_test.go` | Tests for notarize execution pipe |

### Modified Files (4)
| File | Change |
|------|--------|
| `pkg/sign/codesign.go` | Add `hardenedRuntime bool` parameter to `RunCodesign` |
| `internal/pipe/sign/pipe.go` | Pass Hardened Runtime flag based on notarize config |
| `pkg/pipe/registry.go` | Insert `notarize.Pipe{}` between sign and archive |
| `docs/STATE.md` | Update Phase 4 status |

### Unchanged Files
- `pkg/config/config.go` — `NotarizeConfig` already has `AppleID`, `TeamID`, `Password`
- `pkg/context/context.go` — no new `Artifacts` fields needed (`.app` stapled in-place)
- `internal/pipe/notarize/check.go` — existing validation sufficient
- `internal/pipe/archive/pipe.go` — automatically picks up stapled `.app`
- `pkg/archive/zip.go` — reused as-is for creating temp ZIP

## Verification

1. `go build ./...` — compiles without errors
2. `go test ./...` — all tests pass (new + existing)
3. `go vet ./...` — no issues
4. Manual test on a real Xcode project with valid Apple Developer credentials:
   - `macreleaser check` still runs validation only (no notarization attempted)
   - `macreleaser build` runs build → sign (with Hardened Runtime) → notarize → archive
   - The `.app` is signed with `--options runtime`: `codesign -d --verbose=2 dist/<project>/<version>/<App>.app` shows `flags=0x10000(runtime)`
   - Notarization submission succeeds and ticket is stapled
   - Gatekeeper accepts the app: `spctl --assess --type execute --verbose dist/<project>/<version>/<App>.app`
   - Final `.zip`/`.dmg` contain the stapled `.app`

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| `xcrun` not available | `exec.LookPath` check with `xcode-select --install` hint |
| Invalid Apple ID credentials | Detect `"Unable to authenticate"` and suggest app-specific password from appleid.apple.com |
| Notarization rejected by Apple | Parse `"Invalid"` status, suggest `xcrun notarytool log <id>` for details |
| Notarization takes too long | `--wait` has a built-in ~60 min timeout; log message sets expectation ("may take several minutes") |
| `stapler staple` fails (no ticket) | Detect `"Could not find ticket"` and suggest verifying notarytool succeeded |
| `spctl --assess` rejects app | Clear error suggesting app may not be properly signed or notarized |
| Password leaked in logs | `RunSubmit` passes password via command args (not logged by default); pipe logs do not include credential values |
| Hardened Runtime breaks app | This is a real concern for apps using JIT, DYLD env vars, etc. — but entitlements are out of scope for Phase 4. Error from codesign/notarytool will surface the issue. |
| Tests need real Apple credentials | Pure function tests have no system deps; pipe tests only check guard conditions. Real notarization is verified manually. |

## Notes for Future Phases

- **Phase 5** will upload the notarized `.zip`/`.dmg` to GitHub Releases
- **Milestone 4** will add entitlements file support (`--entitlements <path>`) to the sign pipe for apps that need specific Hardened Runtime exceptions
- **Milestone 4** will add `--keychain-profile` authentication as an alternative to Apple ID
- **Milestone 4** will add configurable timeout for `notarytool submit`
