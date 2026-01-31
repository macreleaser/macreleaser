# MacReleaser Phase 6: Basic Cask Generation and Custom Tap Support — Implementation Plan

## Overview

Phase 6 adds Homebrew cask generation and custom tap publishing to the pipeline. After the release pipe uploads archives to GitHub, the homebrew pipe generates a Homebrew cask `.rb` file from the release metadata, computes the SHA256 hash of the archive, and commits the cask to a custom tap repository via the GitHub Contents API. This completes the Milestone 1 steel thread — a fully automated pipeline from `xcodebuild` to an installable Homebrew cask.

## Scope

### In Scope
- Homebrew execution pipe that generates a cask Ruby file from config and release metadata
- SHA256 hash computation of the archive package (pure Go, `crypto/sha256`)
- Cask template rendering via Go `text/template` with version, URL, SHA256, name, desc, homepage, and app stanza
- Asset download URL construction from release config, version, and filename
- Package selection: prefer `.zip`, fall back to `.dmg` (both work with cask `app` stanza)
- GitHub Contents API integration for direct commits to custom tap repositories
- Separate GitHub client for tap authentication (tap token may differ from `GITHUB_TOKEN`)
- Idempotent tap commits: create the cask file if it doesn't exist, update it if it does
- Local cask file generation in the build output directory (always, regardless of tap config)
- Skip publishing for `build` and `snapshot` commands via existing `SkipPublish` context flag
- Unit tests for SHA256 computation, cask rendering, and pipe behavior with mock client

### Out of Scope
- Official Homebrew tap PRs via fork-and-PR workflow (deferred to Milestone 4)
- Cask customization (depends, conflicts, zap stanza, auto_updates) (deferred to Milestone 4)
- Dependency detection
- Multiple cask files per release
- Cask auditing or linting
- Custom branch targeting (always commits to the tap repo's default branch)

## Technical Decisions

### Pipeline Order

The homebrew pipe executes after release, as the final step in the pipeline. It consumes the packages and release metadata produced by the archive and release pipes.

Current `ExecutionPipes`:
```go
var ExecutionPipes = []Piper{
    build.Pipe{},      // Build and extract .app
    sign.Pipe{},       // Code sign with Hardened Runtime
    notarize.Pipe{},   // Submit, wait, staple .app
    archive.Pipe{},    // Package stapled .app into zip/dmg
    release.Pipe{},    // Create GitHub release and upload assets
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
    homebrew.Pipe{},   // Generate cask and commit to tap
}
```

### Cask File Format

A standard Homebrew cask file for a macOS app:

```ruby
cask "myapp" do
  version "1.2.3"
  sha256 "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"

  url "https://github.com/owner/repo/releases/download/v1.2.3/MyApp-1.2.3.zip"
  name "MyApp"
  desc "My awesome macOS application"
  homepage "https://example.com"

  app "MyApp.app"
end
```

The template uses Go `text/template` to render this from a `CaskData` struct. The template is kept as a constant string — no external template files.

### Package Selection

Homebrew casks support both `.zip` and `.dmg` as containers for `.app` bundles. The pipe selects the archive to use:

1. Look for the first `.zip` in `ctx.Artifacts.Packages` → use it
2. If no `.zip`, look for the first `.dmg` → use it
3. If neither found, return an error

`.zip` is preferred because it's the most common cask format and avoids DMG-specific issues (mounting, EULA dialogs). Both formats work identically with the `app` stanza — Homebrew extracts the `.app` automatically.

### Asset Download URL

The cask `url` field needs the direct download URL for the asset, not the release page URL. GitHub release asset URLs follow a deterministic pattern:

```
https://github.com/<owner>/<repo>/releases/download/<tag>/<filename>
```

This is constructed from known values without additional API calls:
- `ctx.Config.Release.GitHub.Owner`
- `ctx.Config.Release.GitHub.Repo`
- `ctx.Version` (the git tag)
- `filepath.Base(packagePath)` (the archive filename)

### App Name Derivation

The `app` stanza needs the `.app` bundle name inside the archive. This is available from `ctx.Artifacts.AppPath`, which was set by the build pipe:

```go
appName := filepath.Base(ctx.Artifacts.AppPath) // e.g., "MyApp.app"
```

### SHA256 Computation

Pure Go implementation using `crypto/sha256` and `io.Copy` for streaming. This avoids shelling out to `shasum` and has no system dependencies. The hash is computed on the selected archive file (the same file the cask URL points to).

### Version Format

Homebrew cask `version` uses the bare version number without the `v` prefix. If `ctx.Version` starts with `v`, it's stripped:

```go
version := strings.TrimPrefix(ctx.Version, "v") // "v1.2.3" → "1.2.3"
```

### GitHub Contents API

The pipe needs to create or update files in the tap repository. Three new methods on `ClientInterface`:

```go
GetFileContents(ctx context.Context, owner, repo, path string) (*github.RepositoryContent, error)
CreateFile(ctx context.Context, owner, repo, path, message string, content []byte) error
UpdateFile(ctx context.Context, owner, repo, path, message string, content []byte, sha string) error
```

- `GetFileContents` checks if the cask file already exists and retrieves its SHA (needed for updates)
- `CreateFile` creates a new file via the Contents API
- `UpdateFile` updates an existing file (requires the current file's SHA to prevent conflicts)

The pipe implements a create-or-update pattern:
1. Call `GetFileContents` for the cask file path
2. If the file exists, call `UpdateFile` with the current SHA
3. If the file doesn't exist (404), call `CreateFile`

This gives idempotent behavior — re-running the release for the same version updates the cask rather than failing.

### Tap Authentication

The tap token (`homebrew.tap.token`) may differ from the `GITHUB_TOKEN` used for releases. The tap is a separate repository that may require different permissions. The homebrew pipe creates its own GitHub client from the tap token, stored on `ctx.HomebrewClient` for test injection.

If no custom tap is configured (all `homebrew.tap` fields empty), the pipe skips the tap commit and only generates the cask file locally.

### Token Validation

Like the release pipe, the tap token is validated at runtime in `Pipe.Run()`, not in `CheckPipe`. This avoids requiring a tap token for `check`, `build`, and `snapshot` commands. `CheckPipe` validates the token field is non-empty only when tap fields are configured — but the actual token value is used at runtime.

### SkipPublish Behavior

The homebrew pipe has two levels of skip:
1. `ctx.SkipPublish` is true → skip the entire pipe (no cask generation, no tap commit)
2. No custom tap configured → generate cask file locally but skip tap commit

This matches the release pipe's behavior with `SkipPublish` and provides useful output even for `build`/`snapshot` by generating the cask file in Phase 6... actually, for consistency with the release pipe, we skip entirely when `SkipPublish` is true. Local cask generation without publish doesn't add enough value to justify the complexity.

### Cask File Path in Tap

The cask file is committed at `Casks/<name>.rb` in the tap repository. This follows the Homebrew convention for cask-only taps.

### Commit Message

- New cask: `"Add <name> <version>"`
- Updated cask: `"Update <name> to <version>"`

### Local Cask File

The cask file is always written to the build output directory at `<BuildOutputDir>/<name>.rb`, regardless of whether a tap commit happens. This allows users to inspect the generated cask and is useful for debugging.

### License Field

The `CaskConfig.License` field is optional. If set, it's included in the cask as a `license` stanza. If empty, the stanza is omitted. This is a simple string (e.g., `"mit"`, `"apache-2.0"`).

### No Config Changes

`HomebrewConfig`, `TapConfig`, and `CaskConfig` already have all required fields. No new config fields are needed for Phase 6.

### Context Changes

One addition to `Context`:
1. `HomebrewClient github.ClientInterface` — injectable GitHub API client for tap operations

One addition to `Artifacts`:
1. `HomebrewCaskPath string` — local path to the generated cask `.rb` file

## Detailed Implementation Tasks

### Task 6.0: Create `pkg/homebrew/sha256.go` — SHA256 Hash Computation

**Subtasks:**

6.0.1. Implement `ComputeSHA256(filePath string) (string, error)` — streams the file through `crypto/sha256` and returns the lowercase hex-encoded hash:
```go
func ComputeSHA256(filePath string) (string, error) {
    f, err := os.Open(filePath)
    if err != nil {
        return "", fmt.Errorf("failed to open file for hashing: %w", err)
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", fmt.Errorf("failed to compute SHA256: %w", err)
    }

    return fmt.Sprintf("%x", h.Sum(nil)), nil
}
```

### Task 6.1: Create `pkg/homebrew/cask.go` — Cask Template Rendering

**Subtasks:**

6.1.1. Define `CaskData` struct containing all fields needed for template rendering:
```go
type CaskData struct {
    Token    string // cask token/identifier (e.g., "myapp")
    Version  string // bare version without v prefix (e.g., "1.2.3")
    SHA256   string // hex-encoded SHA256 hash
    URL      string // direct download URL for the archive
    Name     string // human-readable app name (e.g., "MyApp")
    Desc     string // short description
    Homepage string // homepage URL
    AppName  string // .app bundle name (e.g., "MyApp.app")
    License  string // optional SPDX license identifier
}
```

6.1.2. Define the cask template as a `const`:
```go
const caskTemplate = `cask "{{.Token}}" do
  version "{{.Version}}"
  sha256 "{{.SHA256}}"

  url "{{.URL}}"
  name "{{.Name}}"
  desc "{{.Desc}}"
  homepage "{{.Homepage}}"
{{if .License}}
  license "{{.License}}"
{{end}}
  app "{{.AppName}}"
end
`
```

6.1.3. Implement `RenderCask(data CaskData) (string, error)` — parses and executes the template, returning the rendered cask file content:
```go
func RenderCask(data CaskData) (string, error) {
    tmpl, err := template.New("cask").Parse(caskTemplate)
    if err != nil {
        return "", fmt.Errorf("failed to parse cask template: %w", err)
    }

    var buf bytes.Buffer
    if err := tmpl.Execute(&buf, data); err != nil {
        return "", fmt.Errorf("failed to render cask template: %w", err)
    }

    return buf.String(), nil
}
```

6.1.4. Implement `BuildAssetURL(owner, repo, tag, filename string) string` — constructs the GitHub release asset download URL:
```go
func BuildAssetURL(owner, repo, tag, filename string) string {
    return fmt.Sprintf("https://github.com/%s/%s/releases/download/%s/%s",
        owner, repo, tag, filename)
}
```

6.1.5. Implement `SelectPackage(packages []string) (string, error)` — selects the preferred archive from the package list:
```go
func SelectPackage(packages []string) (string, error) {
    // Prefer .zip, fall back to .dmg
    for _, p := range packages {
        if filepath.Ext(p) == ".zip" {
            return p, nil
        }
    }
    for _, p := range packages {
        if filepath.Ext(p) == ".dmg" {
            return p, nil
        }
    }
    return "", fmt.Errorf("no .zip or .dmg package found for Homebrew cask — ensure archive formats include zip or dmg")
}
```

### Task 6.2: Update `pkg/github/client.go` — Add Contents API to ClientInterface

**Subtasks:**

6.2.1. Add three new methods to `ClientInterface`:
```go
type ClientInterface interface {
    // ... existing methods ...

    GetFileContents(ctx context.Context, owner, repo, path string) (*github.RepositoryContent, error)
    CreateFile(ctx context.Context, owner, repo, path, message string, content []byte) error
    UpdateFile(ctx context.Context, owner, repo, path, message string, content []byte, sha string) error
}
```

6.2.2. Implement `GetFileContents` on `Client`:
```go
func (c *Client) GetFileContents(ctx context.Context, owner, repo, path string) (*github.RepositoryContent, error) {
    content, _, _, err := c.client.Repositories.GetContents(ctx, owner, repo, path, nil)
    if err != nil {
        return nil, fmt.Errorf("failed to get contents of %s in %s/%s: %w", path, owner, repo, err)
    }
    return content, nil
}
```

6.2.3. Implement `CreateFile` on `Client`:
```go
func (c *Client) CreateFile(ctx context.Context, owner, repo, path, message string, content []byte) error {
    opts := &github.RepositoryContentFileOptions{
        Message: &message,
        Content: content,
    }
    _, _, err := c.client.Repositories.CreateFile(ctx, owner, repo, path, opts)
    if err != nil {
        return fmt.Errorf("failed to create file %s in %s/%s: %w", path, owner, repo, err)
    }
    return nil
}
```

6.2.4. Implement `UpdateFile` on `Client`:
```go
func (c *Client) UpdateFile(ctx context.Context, owner, repo, path, message string, content []byte, sha string) error {
    opts := &github.RepositoryContentFileOptions{
        Message: &message,
        Content: content,
        SHA:     &sha,
    }
    _, _, err := c.client.Repositories.UpdateFile(ctx, owner, repo, path, opts)
    if err != nil {
        return fmt.Errorf("failed to update file %s in %s/%s: %w", path, owner, repo, err)
    }
    return nil
}
```

### Task 6.3: Update `pkg/github/mock_client.go` — Add Contents API Mocks

**Subtasks:**

6.3.1. Add tracking fields to `MockClient`:
```go
type MockClient struct {
    // ... existing fields ...

    FileContents    map[string]*github.RepositoryContent // key: "owner/repo/path"
    CreatedFiles    map[string][]byte                    // key: "owner/repo/path", value: content
    UpdatedFiles    map[string][]byte                    // key: "owner/repo/path", value: content
    ContentsError   error                                // if non-nil, returned by GetFileContents
}
```

6.3.2. Update `NewMockClient` to initialize the new maps:
```go
func NewMockClient() *MockClient {
    return &MockClient{
        // ... existing fields ...
        FileContents: make(map[string]*github.RepositoryContent),
        CreatedFiles: make(map[string][]byte),
        UpdatedFiles: make(map[string][]byte),
    }
}
```

6.3.3. Implement `GetFileContents` on `MockClient`:
```go
func (m *MockClient) GetFileContents(ctx context.Context, owner, repo, path string) (*github.RepositoryContent, error) {
    if m.ContentsError != nil {
        return nil, m.ContentsError
    }
    if m.ErrorToReturn != nil {
        return nil, m.ErrorToReturn
    }

    key := fmt.Sprintf("%s/%s/%s", owner, repo, path)
    content, exists := m.FileContents[key]
    if !exists {
        return nil, fmt.Errorf("file %s not found in %s/%s: 404 Not Found", path, owner, repo)
    }
    return content, nil
}
```

6.3.4. Implement `CreateFile` on `MockClient`:
```go
func (m *MockClient) CreateFile(ctx context.Context, owner, repo, path, message string, content []byte) error {
    if m.ErrorToReturn != nil {
        return m.ErrorToReturn
    }

    key := fmt.Sprintf("%s/%s/%s", owner, repo, path)
    m.CreatedFiles[key] = content
    return nil
}
```

6.3.5. Implement `UpdateFile` on `MockClient`:
```go
func (m *MockClient) UpdateFile(ctx context.Context, owner, repo, path, message string, content []byte, sha string) error {
    if m.ErrorToReturn != nil {
        return m.ErrorToReturn
    }

    key := fmt.Sprintf("%s/%s/%s", owner, repo, path)
    m.UpdatedFiles[key] = content
    return nil
}
```

6.3.6. Add helper `AddFileContent` for test setup:
```go
func (m *MockClient) AddFileContent(owner, repo, path string, content *github.RepositoryContent) {
    key := fmt.Sprintf("%s/%s/%s", owner, repo, path)
    m.FileContents[key] = content
}
```

### Task 6.4: Update `pkg/context/context.go` — Add HomebrewClient and HomebrewCaskPath

**Subtasks:**

6.4.1. Add `HomebrewClient` field to `Context`:
```go
type Context struct {
    StdCtx         context.Context
    Config         *config.Config
    Logger         *logrus.Logger
    Version        string
    Artifacts      *Artifacts
    SkipPublish    bool
    GitHubClient   github.ClientInterface
    HomebrewClient github.ClientInterface // injectable GitHub client for tap operations
}
```

6.4.2. Add `HomebrewCaskPath` field to `Artifacts`:
```go
type Artifacts struct {
    BuildOutputDir   string
    ArchivePath      string
    AppPath          string
    Packages         []string
    ReleaseURL       string
    HomebrewCaskPath string // local path to the generated cask .rb file
}
```

### Task 6.5: Create `internal/pipe/homebrew/pipe.go` — Homebrew Execution Pipe

**Subtasks:**

6.5.1. Define `Pipe` struct with `String()` returning `"generating Homebrew cask"`.

6.5.2. Define the `skipError` type following the pattern in `internal/pipe/release/pipe.go`:
```go
type skipError string

func (e skipError) Error() string { return string(e) }
func (e skipError) IsSkip() bool  { return true }
```

6.5.3. Implement `Run(ctx *context.Context) error` with the following steps:

1. **Guard — SkipPublish**: If `ctx.SkipPublish` is true, return `skipError("homebrew publishing skipped")`.

2. **Guard — No packages**: If `len(ctx.Artifacts.Packages) == 0`, return error `"no packages found for Homebrew cask — ensure the archive step completed successfully"`.

3. **Guard — No app path**: If `ctx.Artifacts.AppPath == ""`, return error `"no .app path found — ensure the build step completed successfully"`.

4. **Select package**: Call `homebrew.SelectPackage(ctx.Artifacts.Packages)` to pick the `.zip` or `.dmg` archive. On error, return it.

5. **Compute SHA256**: Call `homebrew.ComputeSHA256(packagePath)`. Log: `"Computing SHA256 hash of <filename>"`. On error, return wrapped error.

6. **Build asset URL**: Call `homebrew.BuildAssetURL(owner, repo, ctx.Version, filename)`.

7. **Build CaskData**: Populate the struct from config and computed values:
   ```go
   data := homebrew.CaskData{
       Token:    ctx.Config.Homebrew.Cask.Name,
       Version:  strings.TrimPrefix(ctx.Version, "v"),
       SHA256:   hash,
       URL:      assetURL,
       Name:     ctx.Config.Project.Name,
       Desc:     ctx.Config.Homebrew.Cask.Desc,
       Homepage: ctx.Config.Homebrew.Cask.Homepage,
       AppName:  filepath.Base(ctx.Artifacts.AppPath),
       License:  ctx.Config.Homebrew.Cask.License,
   }
   ```

8. **Render cask**: Call `homebrew.RenderCask(data)`. On error, return it.

9. **Write local cask file**: Write the rendered cask to `<BuildOutputDir>/<name>.rb`. Store the path in `ctx.Artifacts.HomebrewCaskPath`. Log: `"Generated cask file: <path>"`.

10. **Commit to tap** (if configured): If `isTapConfigured(ctx.Config.Homebrew.Tap)`:
    - Create GitHub client from tap token if not already injected:
      ```go
      if ctx.HomebrewClient == nil {
          client, err := github.NewClient(ctx.Config.Homebrew.Tap.Token)
          // ...
          ctx.HomebrewClient = client
      }
      ```
    - Determine cask file path in tap: `caskPath := fmt.Sprintf("Casks/%s.rb", ctx.Config.Homebrew.Cask.Name)`
    - Read tap config: `tapOwner := ctx.Config.Homebrew.Tap.Owner`, `tapName := ctx.Config.Homebrew.Tap.Name`
    - Try `ctx.HomebrewClient.GetFileContents(ctx.StdCtx, tapOwner, tapName, caskPath)`:
      - If file exists: call `UpdateFile` with the existing SHA and message `"Update <name> to <version>"`
      - If file doesn't exist (error contains `"404"`): call `CreateFile` with message `"Add <name> <version>"`
      - On other errors from `GetFileContents`: return wrapped error
    - On create/update error: return `"failed to commit cask to tap <owner>/<name>: <err>"`
    - Log: `"Committed cask to <tapOwner>/<tapName>: <caskPath>"`

11. **Log success**: `"Homebrew cask generated: <cask_name>"`

### Task 6.6: Register Homebrew Pipe in `pkg/pipe/registry.go`

6.6.1. Add `homebrew.Pipe{}` after `release.Pipe{}` in `ExecutionPipes`:
```go
var ExecutionPipes = []Piper{
    build.Pipe{},      // Build and archive with xcodebuild
    sign.Pipe{},       // Code sign with Hardened Runtime
    notarize.Pipe{},   // Submit, wait, staple .app
    archive.Pipe{},    // Package stapled .app into zip/dmg
    release.Pipe{},    // Create GitHub release and upload assets
    homebrew.Pipe{},   // Generate cask and commit to tap
}
```

The import for `homebrew` already exists (used by `homebrew.CheckPipe{}`).

### Task 6.7: Update `pkg/cli/shared.go` — Print Cask Path in Artifact Summary

6.7.1. Add cask file path to `printArtifactSummary`:
```go
if ctx.Artifacts.HomebrewCaskPath != "" {
    ctx.Logger.Infof("  Cask: %s", ctx.Artifacts.HomebrewCaskPath)
}
```

### Task 6.8: Tests for `pkg/homebrew/sha256_test.go`

Table-driven tests for `ComputeSHA256`:

6.8.1. Create a temp file with known content, verify the hash matches the expected value (pre-computed with `echo -n "content" | shasum -a 256`).

6.8.2. Test with an empty file — verify hash equals SHA256 of empty input (`e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855`).

6.8.3. Test with a non-existent file — verify error contains `"failed to open file"`.

### Task 6.9: Tests for `pkg/homebrew/cask_test.go`

6.9.1. `TestRenderCask` — provide a complete `CaskData` struct, verify the output matches expected Ruby cask format with correct version, sha256, url, name, desc, homepage, and app stanza.

6.9.2. `TestRenderCaskWithLicense` — provide `CaskData` with `License` set, verify the `license` stanza is included.

6.9.3. `TestRenderCaskWithoutLicense` — provide `CaskData` with empty `License`, verify no `license` stanza in output.

6.9.4. `TestBuildAssetURL` — table-driven tests verifying correct URL construction:
- `("owner", "repo", "v1.2.3", "App-1.2.3.zip")` → `"https://github.com/owner/repo/releases/download/v1.2.3/App-1.2.3.zip"`

6.9.5. `TestSelectPackage` — table-driven tests:
- `[".zip", ".dmg"]` → selects `.zip`
- `[".dmg"]` → selects `.dmg`
- `[".dmg", ".zip"]` → selects `.zip` (preference, not order)
- `[".app"]` → error containing `"no .zip or .dmg"`
- `[]` → error containing `"no .zip or .dmg"`

### Task 6.10: Tests for `internal/pipe/homebrew/pipe_test.go`

Following the pattern in `internal/pipe/release/pipe_test.go`:

6.10.1. `TestPipeString` — verify `"generating Homebrew cask"`.

6.10.2. `TestPipeSkipPublish` — set `ctx.SkipPublish = true`, verify the returned error satisfies `pipe.IsSkip`.

6.10.3. `TestPipeNoPackages` — empty `Packages`, verify error containing `"no packages found for Homebrew cask"`.

6.10.4. `TestPipeNoAppPath` — empty `AppPath` with packages set, verify error containing `"no .app path found"`.

6.10.5. `TestPipeGenerateCaskAndCommitToTap` — full happy path:
- Configure homebrew cask and tap fields
- Create a temp file as a stand-in `.zip` package
- Set `ctx.Artifacts.AppPath` to a fake `.app` path
- Inject `MockClient` as `ctx.HomebrewClient`
- Set `ContentsError` to simulate 404 (file doesn't exist yet)
- Verify:
  - No error returned
  - `ctx.Artifacts.HomebrewCaskPath` is set and the file exists on disk
  - The local cask file contains expected content (correct version, sha256, url)
  - `mock.CreatedFiles` contains the cask file at `Casks/<name>.rb`

6.10.6. `TestPipeUpdateExistingCask` — update path:
- Same setup as 6.10.5 but pre-populate `mock.FileContents` with existing cask content and SHA
- Verify `mock.UpdatedFiles` contains the updated cask (not `CreatedFiles`)

6.10.7. `TestPipeNoTapConfigured` — tap fields empty:
- Verify cask file is generated locally (HomebrewCaskPath is set)
- Verify no mock client interactions (no files created or updated)
- No error returned

6.10.8. `TestPipeCommitToTapError` — inject `MockClient` with `SetError()`, verify error is wrapped with `"failed to commit cask to tap"`.

### Task 6.11: Update `docs/STATE.md`

Update Phase 6 status to complete with the list of deliverables.

## Files Summary

### New Files (4)
| File | Purpose |
|------|---------|
| `pkg/homebrew/sha256.go` | `ComputeSHA256()` — streaming SHA256 hash computation |
| `pkg/homebrew/cask.go` | `CaskData`, `RenderCask()`, `BuildAssetURL()`, `SelectPackage()` |
| `pkg/homebrew/sha256_test.go` | Tests for SHA256 computation |
| `pkg/homebrew/cask_test.go` | Tests for cask rendering, URL building, package selection |

### New Files (2) — Internal Pipes
| File | Purpose |
|------|---------|
| `internal/pipe/homebrew/pipe.go` | Homebrew execution pipe |
| `internal/pipe/homebrew/pipe_test.go` | Tests for homebrew execution pipe |

### Modified Files (6)
| File | Change |
|------|--------|
| `pkg/github/client.go` | Add `GetFileContents`, `CreateFile`, `UpdateFile` to `ClientInterface` and implement on `Client` |
| `pkg/github/mock_client.go` | Add mock implementations for Contents API + tracking fields |
| `pkg/context/context.go` | Add `HomebrewClient` to Context; add `HomebrewCaskPath` to Artifacts |
| `pkg/pipe/registry.go` | Add `homebrew.Pipe{}` to `ExecutionPipes` |
| `pkg/cli/shared.go` | Print cask path in artifact summary |
| `docs/STATE.md` | Update Phase 6 status |

### Unchanged Files
- `pkg/config/config.go` — `HomebrewConfig`, `TapConfig`, `CaskConfig` already have all required fields
- `internal/pipe/homebrew/check.go` — existing validation sufficient (validates cask name/desc/homepage, conditionally validates tap owner/name/token)
- `internal/pipe/homebrew/check_test.go` — existing tests unaffected
- `internal/pipe/release/pipe.go` — no changes needed
- `pkg/pipeline/pipeline.go` — no changes needed; `isSkip` already handles `SkipError`
- `pkg/pipe/pipe.go` — `Skip()` and `SkipError` already exist

## Verification

1. `go build ./...` — compiles without errors
2. `go test ./...` — all tests pass (new + existing)
3. `go vet ./...` — no issues
4. Manual test on a real Xcode project with valid GitHub token and a custom tap:
   - `macreleaser check` — runs validation only (no cask generation, no tap token required)
   - `macreleaser build` — builds, signs, notarizes, archives — no cask generated, logs `"Skipping: homebrew publishing skipped"`
   - `macreleaser snapshot` — same as build — no cask generated
   - `macreleaser release` — full pipeline including cask generation and tap commit
   - Local cask file exists at `dist/<project>/<version>/<name>.rb` with correct content
   - Cask file committed to custom tap at `Casks/<name>.rb`
   - `brew install --cask <tap_owner>/<tap_name>/<cask_name>` installs the app successfully
   - Cask path printed in artifact summary

## Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Tap token not set or invalid | Actionable error message; token validated at runtime, not during `check` |
| Tap token lacks repo write permissions | GitHub API returns 403; pipe wraps with hint about required `repo` scope |
| Tap repository doesn't exist | GitHub API returns 404; pipe wraps with message suggesting to create the tap repo first |
| `Casks/` directory doesn't exist in tap | GitHub Contents API `CreateFile` creates intermediate directories automatically |
| SHA256 mismatch after download | SHA256 is computed on the same file that was uploaded; no mismatch possible unless the archive changes after upload (not a Phase 6 concern) |
| Version format inconsistency (`v` prefix) | `strings.TrimPrefix` ensures cask version never has `v` prefix; URL preserves the original tag |
| No `.zip` or `.dmg` in packages | Clear error message suggesting to configure zip or dmg in archive formats |
| Cask template rendering fails | Template is a compile-time constant; failures indicate a bug, not a config issue |
| Race condition on tap update | `GetFileContents` SHA acts as an optimistic lock; concurrent updates detected by GitHub API |
| `AppPath` empty | Guard check at pipe start with actionable error message |
| Mock client doesn't implement new methods | Compile-time `var _ ClientInterface = (*MockClient)(nil)` assertion catches this |

## Notes for Future Phases

- **Milestone 4** will add official Homebrew tap PR support via the fork-and-PR workflow (using the existing `ForkRepository` and `CreatePullRequest` methods on `ClientInterface`)
- **Milestone 4** will add cask customization: `depends_on`, `conflicts_with`, `zap`, `auto_updates`, `caveats` stanzas
- **Milestone 4** will add dependency detection for common frameworks
- **Milestone 4** will add custom branch targeting for tap commits
- **Milestone 4** will add cask linting via `brew audit --cask`
