# MacReleaser Architecture

This document describes the architecture of MacReleaser and provides guidelines for future code contributions.

## Overview

MacReleaser is a release automation tool for macOS applications, inspired by [GoReleaser](https://goreleaser.com). It follows a **pipe-based architecture** where the release process is composed of discrete, composable steps (pipes) that execute in sequence.

## Core Principles

1. **Pipe-Based Architecture**: Each step in the release process is a pipe that implements a common interface
2. **Explicit Dependencies**: Dependencies flow in one direction (cmd → pkg → internal)
3. **Interface-Driven Design**: Use interfaces to enable testing and polymorphism
4. **Configuration-Driven**: Behavior controlled via `.macreleaser.yaml`
5. **Zero Runtime Dependencies**: Single binary with no external dependencies at runtime

## Directory Structure

```
macreleaser/
├── cmd/macreleaser/          # Application entry point
│   └── main.go               # Thin entry point, delegates to pkg/cli
│
├── pkg/                      # Public API packages
│   ├── archive/              # ZIP and DMG packaging helpers
│   ├── build/                # Workspace detection and xcodebuild helpers
│   ├── cli/                  # CLI commands using cobra
│   ├── config/               # Configuration structs and loader
│   ├── context/              # Shared execution context
│   ├── env/                  # Environment variable handling
│   ├── git/                  # Git state resolution (version, commit, branch, dirty)
│   ├── logging/              # Custom log formatters (BulletFormatter)
│   ├── github/               # GitHub API client + interface
│   ├── homebrew/             # Homebrew cask rendering and SHA256
│   ├── notarize/             # Apple notarization (notarytool, staple, spctl)
│   ├── pipe/                 # Pipe interface and registry
│   ├── pipeline/             # Pipeline execution engine
│   ├── sign/                 # Code signing (codesign, identity validation)
│   ├── validate/             # Configuration validation helpers
│   └── version/              # Version information
│
├── internal/                 # Private implementation
│   └── pipe/                 # Individual pipe implementations
│       ├── archive/          # Archive validation (CheckPipe) + ZIP/DMG packaging (Pipe)
│       ├── build/            # Build validation (CheckPipe) + xcodebuild execution (Pipe)
│       ├── homebrew/         # Homebrew validation (CheckPipe) + cask generation and tap commit (Pipe)
│       ├── notarize/         # Notarization validation (CheckPipe) + submit, staple, verify (Pipe)
│       ├── project/          # Project configuration validation (CheckPipe only)
│       ├── release/          # Release validation (CheckPipe) + GitHub release and asset upload (Pipe)
│       └── sign/             # Signing validation (CheckPipe) + codesign with Hardened Runtime (Pipe)
│
└── docs/                     # Documentation
    ├── ARCHITECTURE.md       # This file
    ├── PLAN.md               # Implementation plan
    ├── PRD.md                # Product requirements document
    └── STATE.md              # Current implementation state
```

## Key Components

### 1. Pipe Interface

The fundamental abstraction in MacReleaser. Every step in the release process implements the `Piper` interface.

**Location**: `pkg/pipe/pipe.go`

```go
type Piper interface {
    String() string                    // Returns pipe name for logging
    Run(ctx *context.Context) error    // Executes the pipe
}
```

**Why this pattern?**
- **Composability**: Pipes can be added, removed, or reordered easily
- **Testability**: Each pipe is independently testable
- **Observability**: Clear logging of each step
- **Extensibility**: New phases can be added without changing existing code

### 2. Context

Shared state passed to all pipes. Contains configuration, logger, and any state that needs to be shared between pipes.

**Location**: `pkg/context/context.go`

```go
type Context struct {
    StdCtx         context.Context        // Standard context for cancellation support
    Config         *config.Config         // Parsed configuration
    Logger         *logrus.Logger         // Logger for output
    Version        string                 // Derived from git tag
    Git            git.GitInfo            // Resolved git state (commit, branch, tag, dirty, count)
    Clean          bool                   // When true, remove dist/ before building
    Artifacts      *Artifacts             // Populated by execution pipes
    SkipPublish    bool                   // When true, release and homebrew pipes skip publishing
    SkipNotarize   bool                   // When true, notarize pipe skips; sign disables hardened runtime
    GitHubClient   github.ClientInterface // Injectable GitHub API client
    HomebrewClient github.ClientInterface // Injectable GitHub client for tap operations
}
```

**Guidelines:**
- Keep context minimal - only add fields that multiple pipes need
- `Config` is read-only after creation
- `Git` is populated before the pipeline runs and provides commit, branch, tag, dirty state, and commit count. The build pipe uses `Git.CommitCount` for `CURRENT_PROJECT_VERSION`.
- `Clean` is set by the `--clean` flag and causes `dist/` to be removed before the pipeline runs.
- `Artifacts` is the **intentional exception** to the read-only rule: execution pipes write to it (e.g., the build pipe sets `AppPath`, the archive pipe reads it). This is necessary because execution pipes form a chain where each step produces outputs consumed by the next. Validation pipes must **never** write to `Artifacts`.
- Don't use context for communication between validation pipes (validation pipes should be independent)
- Injectable clients (`GitHubClient`, `HomebrewClient`) enable testing without real API calls. The homebrew client is separate because tap operations may use a different token than release operations.

### 3. Pipeline

Executes registered pipes in two stages: validation then execution. Handles error propagation and skip logic.

**Location**: `pkg/pipeline/pipeline.go`

```go
// RunAll executes validation pipes first, then execution pipes.
func RunAll(ctx *context.Context) error {
    if err := RunValidation(ctx); err != nil {
        return err
    }
    return RunExecution(ctx)
}
```

Each pipe is announced as a top-level bullet via `WithField("action", p.String())`. Duration is logged as a sub-bullet when >= 1 second. The overall command duration is tracked in `shared.go` and printed as `<command> succeeded after Xs`.

### 3a. Logging / Output Format

**Location**: `pkg/logging/formatter.go`

In non-debug mode, output uses `BulletFormatter` which produces goreleaser-style hierarchical bullets:

```
  * loading configuration
  * getting and validating git state
  * git state  branch=main commit=4cb72c9 dirty=false tag=v1.2.3
  * building project
    * scheme=TestApp  configuration=Release
    * took: 45s
  * signing application
  * build succeeded after 48s
```

- Entries with `"action"` field -> top-level bullet: `  * <action>`
- Info-level -> sub-bullet: `    * <message>`
- Warn-level -> `    ! <message>`
- Error-level -> `  x <message>`
- Debug mode uses logrus `TextFormatter` with timestamps

### 4. Pipe Registry

Central registry of all pipes, split into validation and execution stages.

**Location**: `pkg/pipe/registry.go`

```go
// ValidationPipes run first to check configuration.
var ValidationPipes = []Piper{
    project.CheckPipe{},  // Validate project config
    build.CheckPipe{},    // Validate build config
    sign.CheckPipe{},     // Validate signing config
    notarize.CheckPipe{}, // Validate notarization config
    archive.CheckPipe{},  // Validate archive config
    release.CheckPipe{},  // Validate release config
    homebrew.CheckPipe{}, // Validate homebrew config
}

// ExecutionPipes run after validation succeeds.
var ExecutionPipes = []Piper{
    build.Pipe{},      // Build and archive with xcodebuild
    sign.Pipe{},       // Code sign with Hardened Runtime
    notarize.Pipe{},   // Submit, wait, staple .app
    archive.Pipe{},    // Package stapled .app into zip/dmg
    release.Pipe{},    // Create GitHub release and upload assets
    homebrew.Pipe{},   // Generate cask and commit to tap
}
```

Each pipe package may contain both a `CheckPipe` (validation) and a `Pipe` (execution). Only `project` has a `CheckPipe` without a corresponding execution `Pipe`.

**To add a new validation pipe:**
1. Create `CheckPipe` in `internal/pipe/<name>/check.go`
2. Add to `ValidationPipes` in `pkg/pipe/registry.go`

**To add a new execution pipe:**
1. Create `Pipe` in `internal/pipe/<name>/pipe.go`
2. Add to `ExecutionPipes` in `pkg/pipe/registry.go` in the appropriate order

## Design Patterns

### 1. Interface Segregation

Define small, focused interfaces:

```go
// GitHub client interface - only methods currently used
type ClientInterface interface {
    GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error)
    GetRelease(ctx context.Context, owner, repo, tag string) (*github.RepositoryRelease, error)
    // ... etc
}
```

### 2. Dependency Injection

Pass dependencies explicitly rather than using global state:

```go
// Good: Dependencies passed explicitly
func runCheck(cmd *cobra.Command, args []string) {
    logger := SetupLogger(GetDebugMode())
    cfg, err := config.LoadConfig(GetConfigPath())
    ctx := macContext.NewContext(context.Background(), cfg, logger)
    if err := pipeline.RunValidation(ctx); err != nil {
        ExitWithErrorf(logger, "Configuration validation failed: %v", err)
    }
}
```

### 3. Error Wrapping

Always wrap errors with context:

```go
// Good: Error includes context
return fmt.Errorf("failed to load config from %s: %w", path, err)

// Bad: Error loses context
return err
```

### 4. Table-Driven Tests

Use table-driven tests for comprehensive coverage:

```go
func TestPipe(t *testing.T) {
    tests := []struct {
        name    string
        config  config.BuildConfig
        wantErr bool
    }{
        {name: "valid", config: validConfig, wantErr: false},
        {name: "missing config", config: emptyConfig, wantErr: true},
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            pipe := Pipe{}
            ctx := &context.Context{Config: &config.Config{Build: tt.config}}
            err := pipe.Run(ctx)
            if (err != nil) != tt.wantErr {
                t.Errorf("Run() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Dependency Flow

Dependencies in MacReleaser follow a strict hierarchy to maintain clean architecture:

```
cmd/ → pkg/ → internal/
```

- **`cmd/`** - Entry points, no business logic
- **`pkg/`** - Public APIs, interfaces, and shared utilities
- **`internal/`** - Private implementations (pipes)

### Cross-Boundary Import Exception

The `pkg/pipe/registry.go` file imports pipe implementations from `internal/pipe/`. This is an **intentional exception** to the typical dependency flow rule.

**Rationale:** The registry serves as the central authority for pipe execution order. It must know about all pipes to register them, requiring imports from `internal/pipe/`.

**Guidelines:**
- This is the **only** allowed cross-boundary import from `pkg/` to `internal/`
- The registry should not import from any other `internal/` packages
- All pipe imports in the registry must be explicit and documented

```go
// pkg/pipe/registry.go
import (
    // Private implementations (exception to dependency flow)
    "github.com/macreleaser/macreleaser/internal/pipe/archive"
    "github.com/macreleaser/macreleaser/internal/pipe/build"
    "github.com/macreleaser/macreleaser/internal/pipe/homebrew"
    "github.com/macreleaser/macreleaser/internal/pipe/notarize"
    "github.com/macreleaser/macreleaser/internal/pipe/project"
    "github.com/macreleaser/macreleaser/internal/pipe/release"
    "github.com/macreleaser/macreleaser/internal/pipe/sign"
)
```

## Adding New Pipes

### Adding a Validation Pipe

#### Step 1: Create the check file

```bash
mkdir -p internal/pipe/<name>
touch internal/pipe/<name>/check.go
```

#### Step 2: Implement CheckPipe

```go
package <name>

import (
    "github.com/macreleaser/macreleaser/pkg/context"
    "github.com/macreleaser/macreleaser/pkg/validate"
)

type CheckPipe struct{}

func (CheckPipe) String() string { return "validating <name> configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
    cfg := ctx.Config.<Section>

    if err := validate.RequiredString(cfg.SomeField, "<section>.<field>"); err != nil {
        return err
    }

    ctx.Logger.Debug("<Name> configuration validated successfully")
    return nil
}
```

#### Step 3: Register in ValidationPipes

In `pkg/pipe/registry.go`, add to `ValidationPipes`.

#### Step 4: Add tests in `internal/pipe/<name>/check_test.go`

### Adding an Execution Pipe

#### Step 1: Create the pipe file

```bash
touch internal/pipe/<name>/pipe.go
```

#### Step 2: Implement Pipe

```go
package <name>

import (
    "fmt"
    "github.com/macreleaser/macreleaser/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string { return "<action description>" }

func (Pipe) Run(ctx *context.Context) error {
    // Read from ctx.Config and ctx.Artifacts
    // Perform work (build, package, sign, etc.)
    // Write results to ctx.Artifacts
    ctx.Artifacts.SomeField = result
    return nil
}
```

#### Step 3: Register in ExecutionPipes

In `pkg/pipe/registry.go`, add to `ExecutionPipes` in the appropriate order.

#### Step 4: Add tests in `internal/pipe/<name>/pipe_test.go`

## Configuration Handling

### Environment Variable Substitution

MacReleaser supports `env(VAR_NAME)` syntax in configuration:

```yaml
notarize:
  apple_id: "env(APPLE_ID)"
  password: "env(APPLE_APP_SPECIFIC_PASSWORD)"
```

**Implementation**: `pkg/env/env.go`

**Tolerant Substitution**: `SubstituteEnvVarsNode()` resolves env vars that are set and leaves missing ones as literals (e.g., the string `"env(MISSING_VAR)"` is kept as-is). This allows config loading to succeed even when some env vars are not set — validation happens later in CheckPipes.

**Deferred Validation**: `env.CheckResolved(value, field string)` checks if a config value still contains unresolved `env(...)` patterns. CheckPipes call this before their normal validation, producing clear errors like `"notarize.password: environment variable APPLE_PASSWORD is not set"`. CheckPipes for skippable pipes (release, homebrew, notarize) have skip guards that bypass this validation when the pipe won't run.

**Guidelines:**
- Always use environment variables for secrets
- Never hardcode credentials
- Substitution happens at config load time; validation happens in CheckPipes
- CheckPipes must call `env.CheckResolved()` on fields likely to use `env()` references

### Configuration Validation

Validation happens in `CheckPipe` structs, not during config loading:

```go
// config.go - Just loads, doesn't validate
func LoadConfig(path string) (*Config, error) {
    // Load YAML, substitute env vars (tolerant of missing vars)
    // Return Config struct
}

// internal/pipe/build/check.go - Validates (with env var checks)
func (CheckPipe) Run(ctx *context.Context) error {
    if err := env.CheckResolved(cfg.Configuration, "build.configuration"); err != nil {
        return err
    }
    if err := validate.RequiredString(cfg.Configuration, "build.configuration"); err != nil {
        return err
    }
    return nil
}
```

**Skip Guards**: CheckPipes for skippable pipes check their corresponding skip flag before validating:

```go
// internal/pipe/notarize/check.go
func (CheckPipe) Run(ctx *context.Context) error {
    if ctx.SkipNotarize {
        return skipError("notarization skipped via --skip-notarize")
    }
    // ... validate fields ...
}
```

This pattern ensures that `env()` references for skipped pipes don't produce errors.

## Testing Guidelines

### Unit Tests

Each pipe should have comprehensive unit tests:

```go
func TestCheckPipe(t *testing.T) {
    tests := []struct {
        name    string
        config  *config.Config
        wantErr bool
    }{
        // Test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.NewContext(context.Background(), tt.config, logrus.New())
            err := CheckPipe{}.Run(ctx)
            // Assert
        })
    }
}
```

### Integration Tests

Test the full pipeline:

```go
func TestPipeline(t *testing.T) {
    cfg := &config.Config{
        // Valid test config
    }
    ctx := context.NewContext(context.Background(), cfg, logrus.New())

    err := pipeline.RunAll(ctx)
    if err != nil {
        t.Errorf("Pipeline failed: %v", err)
    }
}
```

### Mocking

Use the `ClientInterface` for mocking GitHub operations:

```go
// In tests
type MockGitHubClient struct {
    github.ClientInterface
}

func (m *MockGitHubClient) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
    return &github.Repository{Name: github.String("test")}, nil
}
```

## Security Considerations

1. **File Permissions**: Config files written with `0600` (owner only)
2. **Path Validation**: Config paths and workspace references validated to prevent traversal attacks. The `dist/` output directory uses a fixed path (no user-controlled components).
3. **No Secrets in Config**: All secrets via environment variables
4. **Interface Boundaries**: Clear interfaces prevent accidental misuse

## Code Style

### Naming Conventions

- **Packages**: Short, lowercase (e.g., `build`, `sign`, `config`)
- **Interfaces**: End with `er` or descriptive name (e.g., `Piper`, `ClientInterface`)
- **Structs**: PascalCase (e.g., `Pipe`, `Context`, `Config`)
- **Methods**: PascalCase for exported, camelCase for unexported
- **Variables**: Descriptive, avoid single-letter except for loops

### Error Messages

Be clear and actionable:

```go
// Good
return fmt.Errorf("build.configuration is required")

// Bad
return fmt.Errorf("invalid config")
```

### Comments

- Export comments should start with the identifier name
- Explain "why", not "what"
- Keep comments current with code changes

```go
// CheckPipe validates build configuration and ensures required fields are set.
type CheckPipe struct{}

// Pipe executes the Xcode build, producing an .xcarchive and extracting the .app.
type Pipe struct{}
```

## Common Pitfalls

1. **Don't use init() functions** - They make testing harder and create hidden dependencies
2. **Don't use global state** - Pass dependencies explicitly
3. **Don't duplicate logic** - Use the env package for env var handling
4. **Don't over-engineer** - Simple functions are better than complex abstractions
5. **Don't skip tests** - Every pipe must have tests

## Questions?

If you're unsure about:
- Where to place new code → Follow the directory structure above
- Whether to create a new pipe → If it's a distinct step in the release process, yes
- How to test something → Look at existing pipe tests for examples
- Whether to export something → Start unexported, export only when needed

## References

- [GoReleaser Architecture](https://github.com/goreleaser/goreleaser) - Inspiration for this design
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments) - Go style guidelines
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) - Additional Go best practices
