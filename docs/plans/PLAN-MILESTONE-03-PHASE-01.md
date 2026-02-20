# MacReleaser Milestone 3, Phase 1: GitHub Action for Code Signing Setup — Implementation Plan

## Overview

Phase 1 adds a composite GitHub Action embedded in the macreleaser repository that automates macOS code signing certificate setup in CI and installs the macreleaser binary. The primary value is eliminating the error-prone keychain/certificate dance that developers struggle with when setting up macOS CI pipelines. The action is referenced as `macreleaser/macreleaser@v1`.

## Scope

### In Scope
- `action.yml` at the repository root — composite action
- Temporary keychain creation, certificate import, and partition list configuration
- macreleaser binary installation from GitHub releases
- Update release workflow tag pattern from `v*` to `v*.*.*` to avoid conflicts with action major version tags
- Documentation of required secrets and usage examples

### Out of Scope
- Post-step keychain cleanup (composite actions lack `post` lifecycle hooks; GitHub-hosted runners are ephemeral)
- Notarization credential setup (users pass these as env vars to `macreleaser release`)
- Self-hosted runner keychain cleanup (document as a manual step)
- Integration testing with a test Xcode project (Milestone 3, Phase 2)

## Technical Decisions

### Composite Action

A composite action (shell steps in `action.yml`) is the right choice because:
- All operations are `security` CLI calls — no need for Node.js runtime
- Simpler to maintain than a JavaScript action
- No build step or `node_modules` to manage
- The only tradeoff is no native `post` cleanup hook, which is acceptable for ephemeral runners

### Embedded in Main Repo

The action lives at the root of `macreleaser/macreleaser` rather than a separate `setup-macreleaser` repo:
- Single repo to maintain
- Action version tags (`v1`) are independent from release tags (`v1.0.0`)
- Users reference it as `macreleaser/macreleaser@v1`

### Tag Pattern Change

The existing release workflow triggers on `v*` tags. Adding major version tags (`v1`, `v2`) for the action would conflict with this. The release workflow must be narrowed to `v*.*.*` which matches semver tags like `v1.0.0` but not bare `v1`.

### Keychain Setup

The certificate import follows the well-established macOS CI pattern:

1. **Create temporary keychain** at `$RUNNER_TEMP/macreleaser.keychain-db` with a random password
2. **Configure keychain settings** — disable auto-lock with generous timeout
3. **Add to search list** — prepend to the user's keychain search list so `codesign` and `security find-identity` discover the imported identity
4. **Import .p12 certificate** — decode from base64 secret, import with `-T /usr/bin/codesign`
5. **Set key partition list** — the critical `security set-key-partition-list -S apple-tool:,apple:` incantation that allows non-interactive `codesign` access

The base64 encoding approach is standard: users run `base64 < cert.p12 | pbcopy` to prepare the secret.

### Binary Installation

The install step downloads a pre-built macreleaser binary from GitHub releases:
- Uses `gh release download` for simplicity (available on all GitHub-hosted runners)
- Detects runner architecture (`arm64` vs `amd64`) for the correct binary
- Extracts to `/usr/local/bin` so it's on PATH
- `latest` resolves to the most recent release; specific versions are also supported

## Action Inputs

| Input | Required | Default | Description |
|-------|----------|---------|-------------|
| `p12-base64` | yes | — | Base64-encoded `.p12` certificate file |
| `p12-password` | yes | — | Password for the `.p12` file |
| `macreleaser-version` | no | `latest` | macreleaser version to install (e.g., `v0.3.0`) |

## action.yml Structure

```yaml
name: "Setup MacReleaser"
description: "Set up macOS code signing and install MacReleaser"

inputs:
  p12-base64:
    description: "Base64-encoded .p12 certificate file"
    required: true
  p12-password:
    description: "Password for the .p12 file"
    required: true
  macreleaser-version:
    description: "MacReleaser version to install (default: latest)"
    required: false
    default: "latest"

runs:
  using: "composite"
  steps:
    - name: Set up code signing keychain
      shell: bash
      run: |
        set -euo pipefail

        KEYCHAIN_PATH="$RUNNER_TEMP/macreleaser.keychain-db"
        KEYCHAIN_PASSWORD="$(openssl rand -base64 32)"

        # Create and configure temporary keychain
        security create-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_PATH"
        security set-keychain-settings -lut 21600 "$KEYCHAIN_PATH"
        security unlock-keychain -p "$KEYCHAIN_PASSWORD" "$KEYCHAIN_PATH"

        # Add to search list (prepend to existing)
        security list-keychains -d user -s "$KEYCHAIN_PATH" $(security list-keychains -d user | tr -d '"')

        # Import certificate
        CERT_PATH="$RUNNER_TEMP/certificate.p12"
        echo "${{ inputs.p12-base64 }}" | base64 --decode > "$CERT_PATH"
        security import "$CERT_PATH" \
          -k "$KEYCHAIN_PATH" \
          -P "${{ inputs.p12-password }}" \
          -T /usr/bin/codesign \
          -T /usr/bin/security
        rm -f "$CERT_PATH"

        # Allow codesign to access the key without UI prompt
        security set-key-partition-list \
          -S apple-tool:,apple: \
          -s \
          -k "$KEYCHAIN_PASSWORD" \
          "$KEYCHAIN_PATH"

        echo "Code signing keychain configured"

    - name: Install MacReleaser
      shell: bash
      run: |
        set -euo pipefail

        VERSION="${{ inputs.macreleaser-version }}"
        ARCH="$(uname -m)"
        case "$ARCH" in
          arm64) GOARCH="arm64" ;;
          x86_64) GOARCH="amd64" ;;
          *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
        esac

        REPO="macreleaser/macreleaser"

        if [ "$VERSION" = "latest" ]; then
          gh release download --repo "$REPO" --pattern "macreleaser_*_darwin_${GOARCH}.tar.gz" --dir "$RUNNER_TEMP"
        else
          TAG="$VERSION"
          # Ensure tag has v prefix
          [[ "$TAG" == v* ]] || TAG="v$TAG"
          gh release download "$TAG" --repo "$REPO" --pattern "macreleaser_*_darwin_${GOARCH}.tar.gz" --dir "$RUNNER_TEMP"
        fi

        tar -xzf "$RUNNER_TEMP"/macreleaser_*_darwin_${GOARCH}.tar.gz -C /usr/local/bin macreleaser
        echo "Installed macreleaser $(macreleaser --version)"
      env:
        GH_TOKEN: ${{ github.token }}
```

## Release Workflow Change

The existing `.github/workflows/release.yml` tag trigger changes from:

```yaml
on:
  push:
    tags:
      - "v*"
```

to:

```yaml
on:
  push:
    tags:
      - "v*.*.*"
```

This prevents the GoReleaser release workflow from triggering when action major version tags (`v1`, `v2`) are pushed or moved.

## Files Modified

| File | Change |
|------|--------|
| `action.yml` | New file — composite GitHub Action |
| `.github/workflows/release.yml` | Change tag trigger from `v*` to `v*.*.*` |

## Setup Steps (Manual, One-time)

These are manual steps the developer performs once, not automated:

1. Export the Developer ID certificate and private key from Keychain Access as a `.p12` file
2. Base64-encode it: `base64 < cert.p12 | pbcopy`
3. Add `P12_BASE64` secret to the consuming repo (the macOS app project, not the macreleaser repo)
4. Add `P12_PASSWORD` secret (the password used when exporting the `.p12`)
5. After a macreleaser release, create and push the `v1` tag: `git tag -f v1 HEAD && git push origin v1 --force`

## Example Consumer Workflow

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
      - uses: macreleaser/macreleaser@v1
        with:
          p12-base64: ${{ secrets.P12_BASE64 }}
          p12-password: ${{ secrets.P12_PASSWORD }}
      - run: macreleaser release
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
          APPLE_ID: ${{ secrets.APPLE_ID }}
          APPLE_TEAM_ID: ${{ secrets.APPLE_TEAM_ID }}
          APPLE_APP_SPECIFIC_PASSWORD: ${{ secrets.APPLE_APP_SPECIFIC_PASSWORD }}
```

## Verification

1. `action.yml` passes GitHub Actions schema validation (verified by pushing to a branch)
2. Release workflow tag pattern change: push a `v1` tag and confirm GoReleaser does NOT trigger
3. Push a `v0.4.0` tag and confirm GoReleaser DOES trigger
4. Create a test workflow in the testapp repo that uses `macreleaser/macreleaser@<branch>` to validate the action
5. Keychain setup: `security find-identity -v -p codesigning` shows the imported identity
6. Binary install: `macreleaser --version` shows expected version
7. End-to-end: `macreleaser build --skip-notarize` succeeds using the imported identity
