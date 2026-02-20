# MacReleaser Milestone 3, Phase 2: Integration Testing — Implementation Plan

## Overview

Phase 2 adds an integration test workflow to the testapp repository that exercises the GitHub Action from Phase 1 end-to-end. This validates that the keychain setup, binary installation, and a full `macreleaser build` pipeline work correctly in CI. It also adds a release workflow to testapp so that tagged releases are fully automated.

## Scope

### In Scope
- `.github/workflows/test-action.yml` in testapp — CI workflow that tests the action on every push to main and PRs
- `.github/workflows/release.yml` in testapp — release workflow triggered on `v*.*.*` tags
- Validates keychain setup, signing identity discovery, and `macreleaser build --skip-notarize`
- Validates full `macreleaser release` pipeline via tag-triggered release workflow
- Secrets configuration documentation for the testapp repo

### Out of Scope
- Self-hosted runner support or testing
- Notarization in the CI test workflow (too slow for CI; validated in the release workflow)
- Testing on multiple macOS versions or Xcode versions
- Testing the action from a branch ref (only tests from `v1` tag or main branch during development)

## Technical Decisions

### Two Workflows

The testapp gets two separate workflows:

1. **`test-action.yml`** — Runs on every push to main and on PRs. Exercises the action and runs `macreleaser build --skip-notarize` to validate keychain setup and signing without the cost of notarization. This is the fast feedback loop.

2. **`release.yml`** — Runs on `v*.*.*` tag pushes. Uses the action and runs `macreleaser release` with full notarization. This is the real end-to-end validation.

### Action Reference During Development

During initial development, the test workflow references the action from the main branch:

```yaml
uses: macreleaser/macreleaser@main
```

Once Phase 1 is released and the `v1` tag exists, this changes to:

```yaml
uses: macreleaser/macreleaser@v1
```

### Validation Steps

The CI test workflow validates three things:

1. **Keychain setup** — After the action runs, `security find-identity -v -p codesigning` should list the imported Developer ID identity.
2. **Binary installation** — `macreleaser --version` should succeed and print version info.
3. **Build pipeline** — `macreleaser build --skip-notarize` should complete successfully, producing a signed `.app` and `.zip` in `dist/`.

### Runner Selection

Both workflows use `macos-latest`. The build requires Xcode and macOS-only tools (`xcodebuild`, `codesign`), so there is no Linux alternative.

### Xcode Version

GitHub-hosted macOS runners come with multiple Xcode versions. The workflow uses `maxim-lobanov/setup-xcode` to pin a specific version, ensuring reproducible builds. TestApp targets macOS 15.7+, so Xcode 16.x is required.

## Workflow: test-action.yml

```yaml
name: Test Action

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: maxim-lobanov/setup-xcode@v1
        with:
          xcode-version: latest-stable

      - name: Set up MacReleaser
        uses: macreleaser/macreleaser@v1
        with:
          p12-base64: ${{ secrets.P12_BASE64 }}
          p12-password: ${{ secrets.P12_PASSWORD }}

      - name: Verify signing identity
        run: security find-identity -v -p codesigning

      - name: Verify macreleaser installation
        run: macreleaser --version

      - name: Run build pipeline
        run: macreleaser build --skip-notarize --clean
```

## Workflow: release.yml

```yaml
name: Release

on:
  push:
    tags:
      - "v*.*.*"

jobs:
  release:
    runs-on: macos-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: maxim-lobanov/setup-xcode@v1
        with:
          xcode-version: latest-stable

      - uses: macreleaser/macreleaser@v1
        with:
          p12-base64: ${{ secrets.P12_BASE64 }}
          p12-password: ${{ secrets.P12_PASSWORD }}

      - run: macreleaser release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          APPLE_ID: ${{ secrets.APPLE_ID }}
          APPLE_TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
          APPLE_APP_SPECIFIC_PASSWORD: ${{ secrets.APPLE_APP_SPECIFIC_PASSWORD }}
          HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_TOKEN }}
```

## Files Modified

| File | Repo | Change |
|------|------|--------|
| `.github/workflows/test-action.yml` | testapp | New file — CI workflow to test the action |
| `.github/workflows/release.yml` | testapp | New file — tag-triggered release workflow |

## Setup Steps (Manual, One-time)

These are manual steps performed once in the testapp repo's GitHub settings:

1. Add `P12_BASE64` secret — base64-encoded Developer ID `.p12` certificate
2. Add `P12_PASSWORD` secret — password for the `.p12` file
3. Add `APPLE_ID` secret — Apple ID email for notarization
4. Add `APPLE_TEAM_ID` secret — Apple Developer Team ID
5. Add `APPLE_APP_SPECIFIC_PASSWORD` secret — app-specific password for notarization
6. Add `HOMEBREW_TAP_TOKEN` secret — PAT with repo scope for the homebrew-test-tap repo

## Verification

1. Push to a branch in testapp, open a PR — `test-action.yml` triggers and passes
2. Verify logs show: identity imported, macreleaser version printed, build completes with signed `.app` and `.zip`
3. Create and push a test tag: `git tag v0.1.0 && git push origin v0.1.0`
4. `release.yml` triggers — full pipeline including notarization, GitHub release, and Homebrew cask update
5. GitHub release appears at `github.com/macreleaser/testapp/releases` with the `.zip` asset
6. Homebrew cask updated in `macreleaser/homebrew-test-tap`
7. Switch action reference from `@main` to `@v1` once the action tag is published
