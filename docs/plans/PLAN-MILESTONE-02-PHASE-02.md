# MacReleaser Milestone 2, Phase 2: CI Workflow — Implementation Plan

## Overview

Phase 2 adds a GitHub Actions CI workflow that runs on every push to `main` and on pull requests. It lints, vets, and tests the Go codebase. Since the unit tests mock all macOS system calls, they run on `ubuntu-latest` for speed and cost. A separate macOS job runs smoke tests — building macreleaser and exercising `init` and `check` commands — to validate the binary works on the target platform.

## Scope

### In Scope
- `.github/workflows/ci.yml` workflow file
- Linting with `golangci-lint` via official GitHub Action
- `go vet ./...` and `go test ./...`
- Go module caching via `actions/setup-go` built-in cache
- macOS smoke test job: build macreleaser, run `init` and `check`
- `.golangci.yml` with a minimal, sensible linter configuration

### Out of Scope
- Release workflow (Phase 3)
- Full Xcode integration test (building testapp with macreleaser in CI)
- Code coverage reporting to external services
- Multiple Go version matrix (single stable version)

## Technical Decisions

### Runner Selection

The Go unit tests mock system commands (`xcodebuild`, `codesign`, etc.) and test argument construction, not actual execution. They have no macOS dependency and run on `ubuntu-latest`, which is faster and ~10x cheaper than macOS runners.

A separate `smoke-test` job on `macos-latest` builds the binary and runs non-destructive commands (`macreleaser init`, `macreleaser check`) to verify the binary works on macOS. This catches platform-specific issues (CGO, path handling, etc.) without needing Xcode project fixtures.

### Linter Configuration

Use `golangci-lint` with a conservative set of linters appropriate for the project's stage. Start with defaults plus a few useful linters (`govet`, `errcheck`, `staticcheck`, `unused`, `gosimple`, `ineffassign`). Avoid overly strict linters that would create noise during active development.

### Workflow Triggers

- `push` to `main` — catch regressions immediately
- `pull_request` — validate before merge
- No manual dispatch needed at this stage

### Go Version

Pin to `stable` in `actions/setup-go`, which resolves to the latest stable Go release. The project uses Go 1.24+ features, and `stable` keeps CI current without manual version bumps.

### Concurrency

Use GitHub Actions `concurrency` with `cancel-in-progress: true` for PRs to avoid wasting resources on superseded pushes. Pushes to `main` should not be cancelled.

## Workflow Structure

```yaml
name: CI

on:
  push:
    branches: [main]
  pull_request:

concurrency:
  group: ci-${{ github.ref }}
  cancel-in-progress: ${{ github.ref != 'refs/heads/main' }}

jobs:
  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - uses: golangci/golangci-lint-action@v7
        with:
          version: latest

  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - run: go vet ./...
      - run: go test ./...

  smoke-test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v5
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - run: go build -o macreleaser ./cmd/macreleaser
      - run: ./macreleaser --version
      - run: ./macreleaser init
      - run: ./macreleaser check
```

### Smoke Test Details

The smoke test exercises the happy path without needing signing certs or Apple credentials:

1. `macreleaser --version` — binary runs and prints version
2. `macreleaser init` — generates a `.macreleaser.yaml` in a temp directory
3. `macreleaser check` — validates the generated config

This catches issues like: missing macOS frameworks, broken config defaults, path resolution bugs, and CLI registration errors.

## Files Created

| File | Description |
|------|-------------|
| `.github/workflows/ci.yml` | CI workflow with lint, test, and smoke-test jobs |
| `.golangci.yml` | Linter configuration (minimal, conservative) |

## .golangci.yml Structure

```yaml
linters:
  default: none
  enable:
    - errcheck
    - govet
    - ineffassign
    - staticcheck
    - unused

issues:
  max-issues-per-linter: 0
  max-same-issues: 0
```

## Verification

1. Push a branch and open a PR — CI workflow triggers
2. `lint` job passes (no golangci-lint errors)
3. `test` job passes (`go vet` and `go test` succeed on ubuntu-latest)
4. `smoke-test` job passes (macreleaser builds and runs on macOS)
5. Cancel a superseded PR push — old run is cancelled
6. Push to `main` — CI runs without cancellation
