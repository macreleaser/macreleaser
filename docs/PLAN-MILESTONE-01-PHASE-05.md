# MacReleaser Phase 5: Basic Release to GitHub — Implementation Plan

## Overview

Phase 5 adds GitHub release publishing to the pipeline. After the archive pipe produces `.zip` and/or `.dmg` packages, the release pipe creates a GitHub release tagged with the current version and uploads the packages as release assets. This completes the steel-thread automation from build to public release. The `build` and `snapshot` commands skip publishing; only the `release` command creates a GitHub release.

## Scope

### In Scope
- Release execution pipe that creates a GitHub release via the API
- Upload all archive packages (`ctx.Artifacts.Packages`) as release assets
- GitHub token authentication via `GITHUB_TOKEN` environment variable
- Skip publishing for `build` and `snapshot` commands via `SkipPublish` context flag
- Content type detection for `.zip` and `.dmg` assets
- Filter out non-regular files (e.g., `.app` directories) from upload with a log warning
- Actionable error messages for common failures (missing token, release exists, upload failure)
- Unit tests for content type detection, pipe guard conditions, and mock client interactions

### Out of Scope
- Changelog generation (deferred to Milestone 4)
- Release notes from git history
- Configurable release name format
- Updating or replacing existing releases
- Pre-release flag support
- Draft releases as a separate workflow (the `draft` config field is wired through but not separately tested)

## Technical Decisions

### Pipeline Order

The release pipe executes after archive, as the final step in the pipeline. It consumes the packages produced by the archive pipe.

Current `ExecutionPipes`:
```go
var ExecutionPipes = []Piper{
    build.Pipe{},      // Build and extract .app
    sign.Pipe{},       // Code sign with Hardened Runtime
    notarize.Pipe{},   // Submit, wait, staple .app
    archive.Pipe{},    // Package stapled .app into zip/dmg
}
```

New order:
```go
var ExecutionPipes = []Piper{
    build.Pipe{},      // Build and extract .app
    sign.Pipe{},       // Code sign with Hardened Runtime
    notarize.Pipe{},   // Submit, wait, staple .app
    archive.Pipe{},    // Package stapled .app into zip/dmg
    release.Pipe{},    // Create GitHub release and upload assets
}
```

### SkipPublish Flag

The `build` and `snapshot` commands should not create GitHub releases. Rather than maintaining separate pipeline registries or command-specific pipe lists, a `SkipPublish` flag on `Context` provides a clean signal:

- `build` and `snapshot` commands set `ctx.SkipPublish = true`
- `release` command leaves it as `false` (default)
- The release pipe checks this flag and returns `pipe.Skip("publishing skipped")` when true

This follows GoReleaser's `--skip-publish` pattern and keeps the pipeline registry unified.

The `runPipelineCommand` function gains variadic options to configure the context without changing existing call signatures:

```go
type pipelineOption func(*macContext.Context)

func runPipelineCommand(commandName string, resolveVersion func(*logrus.Logger) string, opts ...pipelineOption)
```

### GitHub Client on Context

The release pipe needs a `github.ClientInterface` for API calls. Adding it to `Context` enables:
- Dependency injection for testing (inject `MockClient`)
- Shared client across pipes (Phase 6 Homebrew will also need it)
- Self-contained client creation — the pipe creates the client from `GITHUB_TOKEN` if not already set on context

The pipe checks `ctx.GitHubClient == nil`, creates one from the environment token, and stores it on context for reuse by subsequent pipes. Tests pre-populate `ctx.GitHubClient` with a mock.

### Token Validation

The `GITHUB_TOKEN` environment variable is validated at the start of `Pipe.Run()`, not in `CheckPipe`. This avoids requiring a token for `check`, `build`, and `snapshot` commands that don't publish. If the token is missing when the release pipe runs (and `SkipPublish` is false), it returns an actionable error.

### Content Type Detection

Asset uploads require a MIME content type. A pure function maps file extensions:
- `.zip` → `application/zip`
- `.dmg` → `application/x-apple-diskimage`
- default → `application/octet-stream`

### Release Name and Tag

- **Tag**: `ctx.Version` (e.g., `v1.2.3`) — the tag must already exist in the remote repository
- **Name**: `"<project.name> <version>"` (e.g., `"MyApp v1.2.3"`)
- **Draft**: Reads `ctx.Config.Release.GitHub.Draft`

### Idempotency

Phase 5 does not check for or update existing releases. If a release already exists for the tag, the GitHub API returns an error, which the pipe wraps with an actionable message suggesting the user delete the existing release or use a different version tag. Upsert behavior is deferred to Milestone 4.

### Non-Regular File Filtering

The archive pipe may include `.app` directories in `ctx.Artifacts.Packages` (when the `app` format is configured). Since `.app` bundles are directories, not regular files, they cannot be uploaded as release assets. The release pipe skips non-regular files with a warning log rather than failing.

### No Config Changes

`ReleaseConfig` and `GitHubConfig` already have all required fields (`Owner`, `Repo`, `Draft`). No new config fields are needed for Phase 5.

### Context Changes

Two additions to `Context`:
1. `SkipPublish bool` — signals whether to skip release publishing
2. `GitHubClient github.ClientInterface` — injectable GitHub API client

One addition to `Artifacts`:
1. `ReleaseURL string` — HTML URL of the created release (used by summary output and Phase 6)

## Detailed Implementation Tasks

### Task 5.0: Update `pkg/context/context.go` — Add SkipPublish, GitHubClient, and ReleaseURL

**Subtasks:**

5.0.1. Add import for `github.com/macreleaser/macreleaser/pkg/github` (no import cycle — `pkg/github` does not import `pkg/context`).

5.0.2. Add `SkipPublish bool` and `GitHubClient github.ClientInterface` fields to `Context`:
```go
type Context struct {
	StdCtx       context.Context
	Config       *config.Config
	Logger       *logrus.Logger
	Version      string
	Artifacts    *Artifacts
	SkipPublish  bool
	GitHubClient github.ClientInterface
}
```

5.0.3. Add `ReleaseURL string` field to `Artifacts`:
```go
type Artifacts struct {
	BuildOutputDir string
	ArchivePath    string
	AppPath        string
	Packages       []string
	ReleaseURL     string
}
```

### Task 5.1: Update `pkg/cli/shared.go` — Add Pipeline Options

**Subtasks:**

5.1.1. Define a `pipelineOption` function type and a `withSkipPublish` constructor:
```go
type pipelineOption func(*macContext.Context)

func withSkipPublish() pipelineOption {
	return func(ctx *macContext.Context) {
		ctx.SkipPublish = true
	}
}
```

5.1.2. Update `runPipelineCommand` signature to accept variadic options and apply them after context creation:
```go
func runPipelineCommand(commandName string, resolveVersion func(*logrus.Logger) string, opts ...pipelineOption) {
	// ... existing setup code ...
	ctx := macContext.NewContext(context.Background(), cfg, logger)
	ctx.Version = version
	for _, opt := range opts {
		opt(ctx)
	}
	// ... existing pipeline execution code ...
}
```

5.1.3. Update `printArtifactSummary` to include the release URL when present:
```go
if ctx.Artifacts.ReleaseURL != "" {
	ctx.Logger.Infof("  Release: %s", ctx.Artifacts.ReleaseURL)
}
```

### Task 5.2: Update `pkg/cli/build.go` — Skip Publishing

5.2.1. Pass `withSkipPublish()` to `runPipelineCommand`:
```go
runPipelineCommand("Build", requireGitVersion, withSkipPublish())
```

### Task 5.3: Update `pkg/cli/snapshot.go` — Skip Publishing

5.3.1. Pass `withSkipPublish()` to `runPipelineCommand`:
```go
runPipelineCommand("Snapshot", snapshotVersion, withSkipPublish())
```

### Task 5.4: Update `pkg/cli/release.go` — Update Description

5.4.1. Update the `Long` description to reflect that release now publishes to GitHub:
```go
Long: `Run the complete release process.
This will build, sign, notarize, package, and release your application
to GitHub. Requires GITHUB_TOKEN environment variable for authentication.`,
```

### Task 5.5: Create `pkg/github/release.go` — Content Type Helper

**Subtasks:**

5.5.1. Implement `ContentTypeForAsset(path string) string` — pure function mapping file extensions to MIME types:
```go
func ContentTypeForAsset(path string) string {
	switch filepath.Ext(path) {
	case ".zip":
		return "application/zip"
	case ".dmg":
		return "application/x-apple-diskimage"
	default:
		return "application/octet-stream"
	}
}
```

### Task 5.6: Create `internal/pipe/release/pipe.go` — Release Execution Pipe

**Subtasks:**

5.6.1. Define `Pipe` struct with `String()` returning `"publishing GitHub release"`.

5.6.2. Implement `Run(ctx *context.Context) error` with the following steps:

1. **Guard — SkipPublish**: If `ctx.SkipPublish` is true, return `pipe.Skip("publishing skipped")`.

2. **Guard — No packages**: If `len(ctx.Artifacts.Packages) == 0`, return error `"no packages to release — ensure the archive step completed successfully"`.

3. **Create GitHub client** if not already injected:
   - Check `ctx.GitHubClient == nil`
   - Call `github.GetGitHubToken()` — if empty, return error: `"GITHUB_TOKEN environment variable is required for publishing — create a token at https://github.com/settings/tokens with 'repo' scope"`
   - Call `github.NewClient(token)` — on error, return wrapped error
   - Store client on `ctx.GitHubClient`

4. **Create release**:
   - Read `owner` and `repo` from `ctx.Config.Release.GitHub`
   - Build release name: `fmt.Sprintf("%s %s", ctx.Config.Project.Name, ctx.Version)`
   - Call `ctx.GitHubClient.CreateRelease()` with `TagName`, `Name`, `Draft`
   - On error containing `"already_exists"`:
     - `"release for tag <version> already exists — delete the existing release or use a different version tag"`
   - On other errors: wrap with `"failed to create GitHub release"`
   - Store `release.GetHTMLURL()` in `ctx.Artifacts.ReleaseURL`
   - Log: `"Created GitHub release: <name>"`

5. **Upload assets** — iterate `ctx.Artifacts.Packages`:
   - `os.Stat(path)` — if error or not a regular file, log warning and skip:
     - `"Skipping %s: not a regular file (only files can be uploaded as release assets)"`
   - Determine content type via `github.ContentTypeForAsset(path)`
   - Call `ctx.GitHubClient.UploadReleaseAsset()` with `release.GetID()`, path, content type
   - On error: return `"failed to upload asset <filename>: <err>"`
   - Log: `"Uploaded: <filename>"`

6. **Log success**: `"Release published: <release_url>"`

### Task 5.7: Update `pkg/github/mock_client.go` — Return Release ID and HTMLURL

5.7.1. Update `CreateRelease` to assign an `ID` and `HTMLURL` on the returned release, so tests can verify the pipe reads them correctly:
```go
func (m *MockClient) CreateRelease(ctx context.Context, owner, repo string, release *github.RepositoryRelease) (*github.RepositoryRelease, error) {
	if m.ErrorToReturn != nil {
		return nil, m.ErrorToReturn
	}
	key := fmt.Sprintf("%s/%s", owner, repo)
	if _, exists := m.Releases[key]; !exists {
		m.Releases[key] = []*github.RepositoryRelease{}
	}
	// Assign synthetic ID and URL for testing
	id := int64(len(m.Releases[key]) + 1)
	release.ID = &id
	htmlURL := fmt.Sprintf("https://github.com/%s/releases/tag/%s", key, release.GetTagName())
	release.HTMLURL = &htmlURL

	m.Releases[key] = append(m.Releases[key], release)
	return release, nil
}
```

### Task 5.8: Register Release Pipe in `pkg/pipe/registry.go`

5.8.1. Add `release.Pipe{}` after `archive.Pipe{}` in `ExecutionPipes`:
```go
var ExecutionPipes = []Piper{
	build.Pipe{},      // Build and archive with xcodebuild
	sign.Pipe{},       // Code sign with Hardened Runtime
	notarize.Pipe{},   // Submit, wait, staple .app
	archive.Pipe{},    // Package stapled .app into zip/dmg
	release.Pipe{},    // Create GitHub release and upload assets
}
```

The import for `release` already exists (used by `release.CheckPipe{}`).

### Task 5.9: Tests for `pkg/github/release_test.go`

Table-driven tests for `ContentTypeForAsset`:
- `"dist/MyApp-1.0.0.zip"` → `"application/zip"`
- `"dist/MyApp-1.0.0.dmg"` → `"application/x-apple-diskimage"`
- `"dist/MyApp-1.0.0.pkg"` → `"application/octet-stream"`
- `"dist/MyApp-1.0.0.tar.gz"` → `"application/octet-stream"`
- `""` → `"application/octet-stream"`

### Task 5.10: Tests for `internal/pipe/release/pipe_test.go`

Following the pattern in `internal/pipe/notarize/pipe_test.go`:

5.10.1. `TestPipeString` — verify `"publishing GitHub release"`

5.10.2. `TestPipeSkipPublish` — set `ctx.SkipPublish = true`, verify the returned error satisfies `pipe.IsSkip`

5.10.3. `TestPipeNoPackages` — empty `Packages`, `SkipPublish = false`, inject mock client → error containing `"no packages to release"`

5.10.4. `TestPipeCreateReleaseAndUpload` — inject `MockClient`, create temp files as stand-in packages, set valid config, verify:
- No error returned
- Release created in mock with correct tag name, release name, and draft flag
- Assets uploaded for each package file
- `ctx.Artifacts.ReleaseURL` is populated with expected URL

5.10.5. `TestPipeCreateReleaseError` — inject `MockClient` with `SetError()`, verify error is wrapped with `"failed to create GitHub release"`

5.10.6. `TestPipeUploadAssetError` — inject `MockClient` that returns error, verify error is wrapped with `"failed to upload asset"`

### Task 5.11: Update `docs/STATE.md`

Update Phase 5 status to complete with the list of deliverables.

## Files Summary

### New Files (3)
| File | Purpose |
|------|---------|
| `pkg/github/release.go` | `ContentTypeForAsset()` pure function |
| `internal/pipe/release/pipe.go` | Release execution pipe |
| `pkg/github/release_test.go` | Tests for content type helper |

### Modified Files (8)
| File | Change |
|------|--------|
| `pkg/context/context.go` | Add `SkipPublish`, `GitHubClient` to Context; add `ReleaseURL` to Artifacts |
| `pkg/cli/shared.go` | Add `pipelineOption` type, `withSkipPublish()`, apply opts in `runPipelineCommand`, print release URL |
| `pkg/cli/build.go` | Pass `withSkipPublish()` |
| `pkg/cli/snapshot.go` | Pass `withSkipPublish()` |
| `pkg/cli/release.go` | Update `Long` description |
| `pkg/github/mock_client.go` | Set `ID` and `HTMLURL` on created releases |
| `pkg/pipe/registry.go` | Add `release.Pipe{}` to `ExecutionPipes` |
| `docs/STATE.md` | Update Phase 5 status |

### New Test File (1)
| File | Purpose |
|------|---------|
| `internal/pipe/release/pipe_test.go` | Tests for release execution pipe |

### Unchanged Files
- `pkg/config/config.go` — `ReleaseConfig` and `GitHubConfig` already have `Owner`, `Repo`, `Draft`
- `internal/pipe/release/check.go` — existing validation sufficient
- `internal/pipe/release/check_test.go` — existing tests unaffected
- `pkg/github/client.go` — existing `CreateRelease()` and `UploadReleaseAsset()` are sufficient
- `pkg/pipeline/pipeline.go` — no changes needed; `isSkip` already handles `SkipError`
- `pkg/pipe/pipe.go` — `Skip()` and `SkipError` already exist

## Verification

1. `go build ./...` — compiles without errors
2. `go test ./...` — all tests pass (new + existing)
3. `go vet ./...` — no issues
4. Manual test on a real Xcode project with valid GitHub token:
   - `macreleaser check` — runs validation only (no GitHub token required)
   - `macreleaser build` — builds, signs, notarizes, archives — no GitHub release created, logs `"Skipping: publishing skipped"`
   - `macreleaser snapshot` — same as build with snapshot version — no GitHub release created
   - `macreleaser release` — full pipeline including GitHub release creation and asset upload
   - GitHub release page shows correct tag, name, and downloadable `.zip`/`.dmg` assets
   - Release URL is printed in the artifact summary

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| `GITHUB_TOKEN` not set | Actionable error with link to token creation page and required `repo` scope |
| Token lacks `repo` scope | GitHub API returns 403; pipe wraps with hint about required permissions |
| Release already exists for tag | Detect `"already_exists"` in API error; suggest deleting existing release |
| Large asset upload fails or times out | GitHub client has 5-minute timeout (set in `NewClient`); error includes filename for context |
| Network interruption during upload | Partial release may exist; user can delete and retry. No automatic cleanup in Phase 5 |
| GitHub API rate limiting | Unlikely for single release + few assets; no special handling needed |
| `.app` directory in Packages | `os.Stat` check filters non-regular files with warning log; upload not attempted |
| Mock doesn't return release ID/URL | Mock enhanced to assign synthetic `ID` and `HTMLURL` on created releases |

## Notes for Future Phases

- **Phase 6** will use `ctx.Artifacts.ReleaseURL` and `ctx.GitHubClient` for Homebrew cask generation and tap commits
- **Milestone 4** will add changelog generation from git history to populate the release body
- **Milestone 4** will add pre-release flag support
- **Milestone 4** will add existing release update/replace behavior (idempotent releases)
