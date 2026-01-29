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
- `macreleaser build` - Build and archive project (Phase 2)
- `macreleaser release` - Full release process including Homebrew
- `macreleaser snapshot` - Test release with snapshot version

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
| `pkg/pipe/registry.go` | Central registry listing all pipes in execution order |
| `pkg/pipeline/pipeline.go` | Pipeline execution engine |
| `pkg/context/context.go` | Shared execution context passed to all pipes |

### CLI Commands
| File | Purpose |
|------|---------|
| `pkg/cli/root.go` | Root command setup |
| `pkg/cli/check.go` | Configuration validation command |
| `pkg/cli/init.go` | Configuration generation command |
| `pkg/cli/build.go` | Build command (Phase 2) |
| `pkg/cli/release.go` | Release command |
| `pkg/cli/snapshot.go` | Snapshot command |
| `pkg/cli/shared.go` | Shared CLI utilities |

### Pipe Implementations (Validation Phase)
All pipes currently validate configuration. Located in `internal/pipe/<name>/pipe.go`:
- `internal/pipe/project/pipe.go` - Project name/scheme validation
- `internal/pipe/build/pipe.go` - Build configuration validation
- `internal/pipe/sign/pipe.go` - Signing identity validation
- `internal/pipe/notarize/pipe.go` - Notarization credentials validation
- `internal/pipe/archive/pipe.go` - Archive format validation
- `internal/pipe/release/pipe.go` - GitHub release config validation
- `internal/pipe/homebrew/pipe.go` - Homebrew cask config validation

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
1. Create `internal/pipe/<name>/pipe.go` implementing the `Piper` interface
2. Add pipe to `pkg/pipe/registry.go` in execution order
3. Add corresponding config struct to `pkg/config/config.go` if needed
4. Add tests in `internal/pipe/<name>/pipe_test.go`

### Pipe Interface
```go
type Piper interface {
    String() string                    // Returns pipe name for logging
    Run(ctx *context.Context) error    // Executes the pipe
}
```

### Configuration Access
Configuration is accessed via the context:
```go
func (Pipe) Run(ctx *context.Context) error {
    cfg := ctx.Config.Build  // Access specific section
    ctx.Logger.Info("message")  // Logging
    return nil
}
```

### Environment Variables
Use `env(VAR_NAME)` syntax in YAML. Substitution happens at config load time via `pkg/env/env.go`.

