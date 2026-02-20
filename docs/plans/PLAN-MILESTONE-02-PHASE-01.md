# MacReleaser Milestone 2, Phase 1: GoReleaser Configuration — Implementation Plan

## Overview

Phase 1 sets up GoReleaser as the build and release tool for macReleaser itself. This includes modifying the version package to support ldflags injection, creating the `.goreleaser.yaml` configuration, and verifying the build locally with `goreleaser build --snapshot`.

## Scope

### In Scope
- Change version package constants to variables (required for `-X` ldflags)
- Create `.goreleaser.yaml` with build configuration for `darwin/amd64` and `darwin/arm64`
- Configure ldflags to inject version, commit hash, and build date
- Configure archive format (tar.gz) with appropriate naming
- Configure changelog generation with category grouping
- Update Makefile to inject version info via ldflags in dev builds
- Update `.gitignore` for GoReleaser artifacts (`/dist/`)
- Verify locally with `goreleaser check` and `goreleaser build --snapshot --clean`

### Out of Scope
- GitHub Actions workflows (Phase 2 and Phase 3)
- Homebrew tap formula configuration (Phase 3)
- Linux or Windows builds (macreleaser is macOS-only)
- GoReleaser Pro features

## Technical Decisions

### Version Package: const to var

GoReleaser uses `-ldflags -X` to inject version information at build time. The `-X` linker flag only works with package-level `var` declarations, not `const`. The version package must change from:

```go
const (
    Version = "dev"
    Commit  = "unknown"
    Date    = "unknown"
)
```

to:

```go
var (
    version = "dev"
    commit  = "unknown"
    date    = "unknown"
)
```

The `Name` constant stays as `const` since it never changes. The variables are lowercase (unexported) to prevent external mutation — the existing `VersionInfo()` and `ShortVersion()` functions remain the public API.

### GoReleaser ldflags

GoReleaser provides template variables for version info. Since our version variables live in `pkg/version` (not `main`), we configure explicit ldflags:

```yaml
ldflags:
  - -s -w
  - -X github.com/macreleaser/macreleaser/pkg/version.version={{.Version}}
  - -X github.com/macreleaser/macreleaser/pkg/version.commit={{.Commit}}
  - -X github.com/macreleaser/macreleaser/pkg/version.date={{.Date}}
```

`-s -w` strips debug info and DWARF tables for smaller binaries.

### darwin-only Builds

MacReleaser shells out to macOS-only tools (`xcodebuild`, `codesign`, `hdiutil`, `xcrun`). Building for Linux or Windows would produce a binary that cannot function. The config restricts `goos` to `darwin` and `goarch` to `amd64` and `arm64`.

### Archive Naming

Archives follow the convention `macreleaser_<version>_<os>_<arch>.tar.gz`. GoReleaser's default naming template handles this well with `{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}`.

### Changelog Categories

GoReleaser can group changelog entries by commit message prefix. We use a simple scheme:
- `feat:` → Features
- `fix:` → Bug Fixes
- `docs:` → Documentation
- Other → Other

### Makefile ldflags

The Makefile `build` target should also inject version info for dev builds, using `git describe --tags` for version and `git rev-parse --short HEAD` for commit. This ensures `macreleaser --version` shows useful info even in local dev builds.

## Files Modified

| File | Change |
|------|--------|
| `pkg/version/version.go` | Change `Version`, `Commit`, `Date` from `const` to `var` (lowercase, unexported); update `VersionInfo()` and `ShortVersion()` to use the vars |
| `.goreleaser.yaml` | New file — GoReleaser configuration |
| `Makefile` | Add ldflags to `build` target for dev version injection |
| `.gitignore` | Ensure `/dist/` is listed (already present, verify) |

## .goreleaser.yaml Structure

```yaml
# yaml-language-server: $schema=https://goreleaser.com/static/schema.json
version: 2

builds:
  - id: macreleaser
    main: ./cmd/macreleaser
    binary: macreleaser
    env:
      - CGO_ENABLED=0
    goos:
      - darwin
    goarch:
      - amd64
      - arm64
    ldflags:
      - -s -w
      - -X github.com/macreleaser/macreleaser/pkg/version.version={{.Version}}
      - -X github.com/macreleaser/macreleaser/pkg/version.commit={{.Commit}}
      - -X github.com/macreleaser/macreleaser/pkg/version.date={{.Date}}

archives:
  - id: macreleaser
    formats:
      - tar.gz

changelog:
  sort: asc
  filters:
    exclude:
      - "^test:"
      - "^chore:"
      - "Merge pull request"
  groups:
    - title: Features
      regexp: '^feat:'
    - title: Bug Fixes
      regexp: '^fix:'
    - title: Documentation
      regexp: '^docs:'
    - title: Other
      order: 999
```

## Verification

1. `go build ./...` — still compiles after version package changes
2. `go test ./...` — all existing tests pass
3. `goreleaser check` — config is valid
4. `goreleaser build --snapshot --clean` — produces binaries in `dist/`
5. `./dist/macreleaser_darwin_arm64/macreleaser --version` — shows snapshot version info
6. `make build && ./bin/macreleaser --version` — shows dev version with git info
