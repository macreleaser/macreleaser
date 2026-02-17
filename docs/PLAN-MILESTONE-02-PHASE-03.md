# MacReleaser Milestone 2, Phase 3: Release Workflow — Implementation Plan

## Overview

Phase 3 adds a GitHub Actions release workflow that triggers on version tag pushes. It uses the GoReleaser action to build darwin/amd64 and darwin/arm64 binaries, create a GitHub release with changelog, and publish a Homebrew formula to the `macreleaser/homebrew-tap` repository for easy installation via `brew install macreleaser/tap/macreleaser`.

## Scope

### In Scope
- `.github/workflows/release.yml` triggered on `v*` tag pushes
- GoReleaser action to build, archive, and create GitHub release
- Homebrew formula generation via GoReleaser `brews` configuration
- `HOMEBREW_TAP_TOKEN` secret for cross-repo tap commits
- Add `brews` section to `.goreleaser.yaml`
- Document the release process in the repo README or a RELEASING.md

### Out of Scope
- Binary signing or notarization of the macreleaser binary itself
- Universal binary (fat binary) — separate arch binaries are sufficient
- Publishing to official Homebrew core
- Automated version bumping or tag creation

## Technical Decisions

### Tag-triggered Release

The workflow triggers on `v*` tag pushes only (e.g., `v0.1.0`). This is the standard GoReleaser pattern. The version is derived from the tag — no manual version editing needed.

### Runner Selection

GoReleaser cross-compiles Go binaries without needing the target OS. Since macreleaser is pure Go with `CGO_ENABLED=0`, darwin binaries build correctly on `ubuntu-latest`. This avoids expensive macOS runner minutes for releases.

### Homebrew Tap

GoReleaser's `brews` section generates a Homebrew formula and commits it to a tap repository. The `macreleaser/homebrew-tap` repo already exists in the workspace. GoReleaser will create/update a `Formula/macreleaser.rb` file there.

This requires a Personal Access Token (PAT) with `repo` scope stored as `HOMEBREW_TAP_TOKEN` in the macreleaser repo's secrets. The default `GITHUB_TOKEN` is scoped to the current repository and cannot push to other repos.

### Token Configuration

Two tokens are used:
- `GITHUB_TOKEN` (automatic) — creates the GitHub release and uploads assets in the macreleaser repo
- `HOMEBREW_TAP_TOKEN` (secret) — pushes the formula to `macreleaser/homebrew-tap`

### Formula Configuration

The Homebrew formula includes:
- `install`: Copies the binary to `bin/`
- `test`: Runs `macreleaser --version`
- `homepage`: Points to the GitHub repo
- `description`: One-line description of macreleaser
- No dependencies (static Go binary)

### Release Notes

GoReleaser generates release notes from the changelog configuration set up in Phase 1. Commits are grouped by type (features, fixes, docs) and filtered to exclude noise (test, chore, merge commits).

## Workflow Structure

```yaml
name: Release

on:
  push:
    tags:
      - "v*"

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0
      - uses: actions/setup-go@v5
        with:
          go-version: stable
      - uses: goreleaser/goreleaser-action@v6
        with:
          version: "~> v2"
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

`fetch-depth: 0` is required — GoReleaser needs full git history for changelog generation and version detection.

## .goreleaser.yaml Additions

The `brews` section added to the existing `.goreleaser.yaml`:

```yaml
brews:
  - name: macreleaser
    repository:
      owner: macreleaser
      name: homebrew-tap
      token: "{{ .Env.HOMEBREW_TAP_TOKEN }}"
    directory: Formula
    homepage: "https://github.com/macreleaser/macreleaser"
    description: "macOS app release automation — build, sign, notarize, and distribute"
    install: |
      bin.install "macreleaser"
    test: |
      system bin/"macreleaser", "--version"
    commit_author:
      name: macreleaser-bot
      email: macreleaser-bot@users.noreply.github.com
```

## Files Modified

| File | Change |
|------|--------|
| `.github/workflows/release.yml` | New file — release workflow |
| `.goreleaser.yaml` | Add `brews` section for Homebrew formula |

## Setup Steps (Manual, One-time)

These are manual steps the developer performs once, not automated:

1. Create a GitHub PAT with `repo` scope
2. Add `HOMEBREW_TAP_TOKEN` secret to the `macreleaser/macreleaser` repo settings
3. Ensure `macreleaser/homebrew-tap` repo has a `Formula/` directory (create if needed)

## Verification

1. `goreleaser check` — config still valid with `brews` section
2. `goreleaser release --snapshot --clean` — snapshot build succeeds locally (no actual release)
3. Create and push a test tag: `git tag v0.1.0 && git push origin v0.1.0`
4. Release workflow triggers and completes
5. GitHub release appears with darwin/amd64 and darwin/arm64 archives
6. Changelog groups commits by type
7. Homebrew formula appears in `macreleaser/homebrew-tap/Formula/macreleaser.rb`
8. `brew install macreleaser/tap/macreleaser` installs the binary
9. `macreleaser --version` shows the released version
