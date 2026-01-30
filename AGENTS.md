# OpenCode Agent Guidelines

## Project Overview

**MacReleaser** is a release automation tool for macOS applications, inspired by GoReleaser but purpose-built for the macOS app ecosystem. It automates the build, sign, notarize, and release process for Developer ID signed macOS applications built using Xcode.

### Key Technologies
- **Language**: Go 1.24+
- **Architecture**: Pipe-based pipeline (each step is a composable "pipe")
- **CLI Framework**: Cobra
- **Configuration**: YAML with environment variable substitution
- **Logging**: Logrus

### Commands
- `macreleaser init` - Generate example configuration
- `macreleaser check` - Validate configuration file
- `macreleaser build` - Build, archive, and package project
- `macreleaser release` - Full release process (signing, notarization, and upload coming soon)
- `macreleaser snapshot` - Test build with snapshot version

## Key Files for Context

When working on this project, these files contain essential context:

### Documentation
| File | Purpose |
|------|---------|
| `README.md` | User-facing documentation, quick start, configuration examples |
| `docs/ARCHITECTURE.md` | Comprehensive architecture guide, design patterns, coding standards |
| `docs/PLAN.md` | Implementation plan |
| `docs/PRD.md` | Product requirements document |
| `docs/STATE.md` | Current implementation status |

### Core Configuration
| File | Purpose |
|------|---------|
| `pkg/config/config.go` | All configuration structs (Config, ProjectConfig, BuildConfig, etc.) |
| `pkg/env/env.go` | Environment variable substitution logic |

### Architecture Components
| File | Purpose |
|------|---------|
| `pkg/pipe/pipe.go` | Piper interface - all pipes implement this |
| `pkg/pipe/registry.go` | Central registry with `ValidationPipes` and `ExecutionPipes` |
| `pkg/pipeline/pipeline.go` | Pipeline execution engine (`RunValidation`, `RunAll`) |
| `pkg/context/context.go` | Shared execution context (Config, Version, Artifacts) |
| `pkg/build/detect.go` | Workspace/project auto-detection |
| `pkg/build/xcodebuild.go` | xcodebuild argument construction and execution |
| `pkg/archive/zip.go` | ZIP packaging via ditto |
| `pkg/archive/dmg.go` | DMG packaging via hdiutil |
| `pkg/git/version.go` | Git tag version resolution |

### CLI Commands
| File | Purpose |
|------|---------|
| `pkg/cli/root.go` | Root command setup |
| `pkg/cli/check.go` | Configuration validation command |
| `pkg/cli/init.go` | Configuration generation command |
| `pkg/cli/build.go` | Build command + `requireGitVersion` helper |
| `pkg/cli/release.go` | Release command |
| `pkg/cli/snapshot.go` | Snapshot command + `snapshotVersion` helper |
| `pkg/cli/shared.go` | Shared CLI utilities (`runPipelineCommand`, `printArtifactSummary`) |

### Pipe Implementations
Each pipe package is in `internal/pipe/<name>/`. Validation logic is in `check.go` (`CheckPipe`), execution logic in `pipe.go` (`Pipe`).

**Validation pipes** (`check.go`):
- `internal/pipe/project/check.go` - Project name/scheme validation
- `internal/pipe/build/check.go` - Build configuration validation
- `internal/pipe/sign/check.go` - Signing identity validation
- `internal/pipe/notarize/check.go` - Notarization credentials validation
- `internal/pipe/archive/check.go` - Archive format validation
- `internal/pipe/release/check.go` - GitHub release config validation
- `internal/pipe/homebrew/check.go` - Homebrew cask config validation

**Execution pipes** (`pipe.go`):
- `internal/pipe/build/pipe.go` - xcodebuild archive + .app extraction
- `internal/pipe/archive/pipe.go` - ZIP/DMG packaging

### External Integrations
| File | Purpose |
|------|---------|
| `pkg/github/client.go` | GitHub API client |
| `pkg/github/mock_client.go` | Mock GitHub client for testing |

### Build & Development
| File | Purpose |
|------|---------|
| `Makefile` | Build, test, and development commands |
| `go.mod` | Go module dependencies |
| `cmd/macreleaser/main.go` | Application entry point |

## Architecture Patterns

### Adding a New Pipe
1. **Validation pipe**: Create `internal/pipe/<name>/check.go` with `CheckPipe` struct, add to `ValidationPipes` in `pkg/pipe/registry.go`
2. **Execution pipe**: Create `internal/pipe/<name>/pipe.go` with `Pipe` struct, add to `ExecutionPipes` in `pkg/pipe/registry.go`
3. Add corresponding config struct to `pkg/config/config.go` if needed
4. Add tests in `internal/pipe/<name>/check_test.go` and/or `pipe_test.go`

### Pipe Interface
```go
type Piper interface {
    String() string                    // Returns pipe name for logging
    Run(ctx *context.Context) error    // Executes the pipe
}
```

### Configuration and Artifacts Access
```go
// Validation pipe — reads config only
func (CheckPipe) Run(ctx *context.Context) error {
    cfg := ctx.Config.Build  // Access specific section
    return nil
}

// Execution pipe — reads config and writes artifacts
func (Pipe) Run(ctx *context.Context) error {
    cfg := ctx.Config.Build
    ctx.Artifacts.AppPath = result  // Populate for next pipe
    return nil
}
```

### Environment Variables
Use `env(VAR_NAME)` syntax in YAML. Substitution happens at config load time via `pkg/env/env.go`.

