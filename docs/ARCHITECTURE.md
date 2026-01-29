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
│   ├── cli/                  # CLI commands using cobra
│   ├── config/               # Configuration structs and loader
│   ├── context/              # Shared execution context
│   ├── env/                  # Environment variable handling
│   ├── github/               # GitHub API client + interface
│   ├── pipe/                 # Pipe interface and registry
│   ├── pipeline/             # Pipeline execution engine
│   └── version/              # Version information
│
├── internal/                 # Private implementation
│   └── pipe/                 # Individual pipe implementations
│       ├── archive/          # Archive configuration validation
│       ├── build/            # Build configuration validation
│       ├── homebrew/         # Homebrew configuration validation
│       ├── notarize/         # Notarization configuration validation
│       ├── project/          # Project configuration validation
│       ├── release/          # Release configuration validation
│       └── sign/             # Signing configuration validation
│
└── docs/                     # Documentation
│   ├── ARCHITECTURE.md       # This file
│   ├── PLAN.md               # Implementation plan
    └── PRD.md                # Product requirements document
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
    Config *config.Config    // Parsed configuration
    Logger *logrus.Logger    // Logger for output
}
```

**Guidelines:**
- Keep context minimal - only add fields that multiple pipes need
- Don't use context for pipe-to-pipe communication (pipes should be independent)
- Context is read-only after creation (except for logging)

### 3. Pipeline

Executes all registered pipes in sequence. Handles error propagation and skip logic.

**Location**: `pkg/pipeline/pipeline.go`

```go
func Run(ctx *context.Context) error {
    for _, p := range pipe.All {
        ctx.Logger.Infof("Running: %s", p.String())
        if err := p.Run(ctx); err != nil {
            if isSkip(err) {
                ctx.Logger.Infof("Skipping: %v", err)
                continue
            }
            return fmt.Errorf("%s: %w", p.String(), err)
        }
    }
    return nil
}
```

### 4. Pipe Registry

Central registry of all pipes in execution order.

**Location**: `pkg/pipe/registry.go`

```go
var All = []Piper{
    project.Pipe{},   // First: validate project config
    build.Pipe{},     // Then: validate build config
    sign.Pipe{},      // Then: validate signing config
    // ... etc
}
```

**To add a new pipe:**
1. Create pipe implementation in `internal/pipe/<name>/pipe.go`
2. Add to `pkg/pipe/registry.go` in the appropriate order
3. Import the package in `registry.go`

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
    ctx := context.New(cfg, logger)
    if err := pipeline.Run(ctx); err != nil {
        ExitWithError(logger, "Validation failed: %v", err)
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
    // Standard library
    
    // Public packages
    
    // Private implementations (exception to dependency flow)
    "github.com/macreleaser/macreleaser/internal/pipe/archive"
    "github.com/macreleaser/macreleaser/internal/pipe/build"
    // ... etc
)
```

## Adding New Pipes

### Step 1: Create the Pipe Package

```bash
mkdir -p internal/pipe/<name>
touch internal/pipe/<name>/pipe.go
```

### Step 2: Implement the Pipe

```go
package <name>

import (
    "fmt"
    "github.com/macreleaser/macreleaser/pkg/context"
)

type Pipe struct{}

func (Pipe) String() string { 
    return "<description of what this pipe does>" 
}

func (Pipe) Run(ctx *context.Context) error {
    // Access config via ctx.Config
    cfg := ctx.Config.<Section>
    
    // Validate or execute
    if cfg.SomeField == "" {
        return fmt.Errorf("<section>.<field> is required")
    }
    
    // Log progress
    ctx.Logger.Debug("<action> completed successfully")
    
    return nil
}
```

### Step 3: Register the Pipe

In `pkg/pipe/registry.go`:

```go
import (
    // ... existing imports
    "github.com/macreleaser/macreleaser/internal/pipe/<name>"
)

var All = []Piper{
    // ... existing pipes
    <name>.Pipe{},  // Add in appropriate order
}
```

### Step 4: Add Tests

```bash
touch internal/pipe/<name>/pipe_test.go
```

## Configuration Handling

### Environment Variable Substitution

MacReleaser supports `env(VAR_NAME)` syntax in configuration:

```yaml
notarize:
  apple_id: "env(APPLE_ID)"
  password: "env(APPLE_APP_SPECIFIC_PASSWORD)"
```

**Implementation**: `pkg/env/env.go`

**Guidelines:**
- Always use environment variables for secrets
- Never hardcode credentials
- The substitution happens at config load time, not runtime

### Configuration Validation

Validation happens in pipes, not during config loading:

```go
// config.go - Just loads, doesn't validate
func LoadConfig(path string) (*Config, error) {
    // Load YAML, substitute env vars
    // Return Config struct
}

// internal/pipe/build/pipe.go - Validates
func (Pipe) Run(ctx *context.Context) error {
    if ctx.Config.Build.Configuration == "" {
        return fmt.Errorf("build.configuration is required")
    }
    return nil
}
```

## Testing Guidelines

### Unit Tests

Each pipe should have comprehensive unit tests:

```go
func TestPipe(t *testing.T) {
    tests := []struct {
        name    string
        config  *config.Config
        wantErr bool
    }{
        // Test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            ctx := context.New(tt.config, logrus.New())
            pipe := Pipe{}
            err := pipe.Run(ctx)
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
    ctx := context.New(cfg, logrus.New())
    
    err := pipeline.Run(ctx)
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
2. **Path Validation**: Config paths validated to prevent traversal attacks
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
// Pipe validates build configuration and ensures required fields are set.
type Pipe struct{}

// Run executes the validation and returns an error if the configuration is invalid.
func (Pipe) Run(ctx *context.Context) error {
    // ...
}
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
