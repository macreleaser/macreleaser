# MacReleaser Phase 1: Core Foundation Implementation Tasks

## Overview

This phase establishes the foundational architecture and CLI framework for macreleaser. No actual build/release operations will be performed - this focuses on structure, configuration parsing, and validation system.

## Technical Decisions Made

### Core Architecture Decisions
- **Pipe Architecture**: Implement goreleaser's pipe-based validation system from the start
- **Environment Variables**: Simple `env(VAR_NAME)` substitution (templates can come later)
- **ID Uniqueness**: Track IDs across all sections from beginning (cross-section uniqueness)
- **YAML Library**: Use `goccy/go-yaml` instead of `gopkg.in/yaml.v3`
- **Validation**: Mixed approach - structured validation for basic fields + custom logic for complex rules
- **Error Patterns**: Follow goreleaser's "what failed + how to fix" message format
- **Logging**: Follow goreleaser's lead on structured logging approach

## Detailed Implementation Tasks

### Task 1.1: Project Structure and Module Setup
**Estimated Time**: 0.5 days

**Subtasks:**
1.1.1 Create directory structure:
```
macreleaser/
├── cmd/
│   └── macreleaser/
│       └── main.go           # CLI entry point
├── pkg/
│   ├── config/              # Configuration handling
│   │   ├── config.go       # Config structs and validation
│   │   ├── config_test.go  
│   │   ├── pipe.go         # Pipe interface and coordinator
│   │   └── validation/     # Section validators
│   │       ├── build.go
│   │       ├── archive.go
│   │       ├── release.go
│   │       ├── homebrew.go
│   │       └── ids.go       # ID uniqueness system
│   ├── cli/                 # CLI command definitions
│   │   ├── root.go         # Root command setup
│   │   ├── init.go         # init command
│   │   ├── check.go        # check command  
│   │   ├── build.go        # build command (placeholder)
│   │   ├── release.go      # release command (placeholder)
│   │   ├── snapshot.go     # snapshot command (placeholder)
│   │   └── shared.go       # Shared CLI utilities
│   ├── env/                  # Environment variable handling
│   │   └── env.go          # Simple env(VAR_NAME) substitution
│   └── version/             # Version handling
│       └── version.go
├── .macreleaser.yaml        # Example configuration
├── go.mod                    # Go module
├── go.sum
├── Makefile                  # Build targets
├── README.md
└── LICENSE
```

1.1.2 Initialize Go module:
```bash
go mod init github.com/macreleaser/macreleaser
```

1.1.3 Add dependencies:
```go
require (
    github.com/spf13/cobra v1.8.0
    github.com/spf13/viper v1.18.2  
    github.com/goccy/go-yaml v1.11.3
    github.com/google/go-github v45.0.0
    github.com/sirupsen/logrus v1.9.3
)
```

1.1.4 Create basic Makefile targets:
```makefile
.PHONY: build test clean install

build:
	go build -o bin/macreleaser ./cmd/macreleaser

test:
	go test ./...

clean:
	rm -rf bin/

install: build
	cp bin/macreleaser /usr/local/bin/

test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
```

### Task 1.2: Core Configuration System
**Estimated Time**: 1.5 days

**Subtasks:**
1.2.1 Define configuration structs (pkg/config/config.go):
```go
type Config struct {
    Project   ProjectConfig   `yaml:"project"`
    Build     BuildConfig     `yaml:"build"`
    Sign      SignConfig      `yaml:"sign"`
    Notarize NotarizeConfig  `yaml:"notarize"`
    Archive   ArchiveConfig   `yaml:"archive"`
    Release   ReleaseConfig   `yaml:"release"`
    Homebrew  HomebrewConfig `yaml:"homebrew"`
}

type ProjectConfig struct {
    Name     string `yaml:"name"`
    Scheme   string `yaml:"scheme"`
    Workspace string `yaml:"workspace,omitempty"`
}

type BuildConfig struct {
    Configuration  string   `yaml:"configuration"`
    Architectures []string `yaml:"architectures"`
}

type SignConfig struct {
    Identity string `yaml:"identity"`
}

type NotarizeConfig struct {
    AppleID string `yaml:"apple_id"`
    TeamID  string `yaml:"team_id"`
    Password string `yaml:"password"`
}

type ArchiveConfig struct {
    Formats []string `yaml:"formats"`
    DMG     DMGConfig `yaml:"dmg,omitempty"`
}

type DMGConfig struct {
    Background string `yaml:"background,omitempty"`
    IconSize  int    `yaml:"icon_size,omitempty"`
}

type ReleaseConfig struct {
    GitHub GitHubConfig `yaml:"github"`
}

type GitHubConfig struct {
    Owner string `yaml:"owner"`
    Repo  string `yaml:"repo"`
    Draft bool   `yaml:"draft"`
}

type HomebrewConfig struct {
    Tap      TapConfig      `yaml:"tap,omitempty"`
    Official OfficialConfig `yaml:"official,omitempty"`
    Cask     CaskConfig     `yaml:"cask"`
}

type TapConfig struct {
    Owner string `yaml:"owner"`
    Name  string `yaml:"name"`
    Token string `yaml:"token"`
}

type OfficialConfig struct {
    Enabled   bool     `yaml:"enabled"`
    Token     string   `yaml:"token"`
    AutoMerge bool     `yaml:"auto_merge"`
    Assignees []string `yaml:"assignees"`
}

type CaskConfig struct {
    Name     string `yaml:"name"`
    Desc     string `yaml:"desc"`
    Homepage string `yaml:"homepage"`
    License  string `yaml:"license"`
}
```

1.2.2 Implement YAML loading with strict parsing:
```go
func LoadConfig(path string) (*Config, error) {
    if path == "" {
        return nil, errors.New("config file path is required")
    }
    
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }
    
    var config Config
    if err := yaml.UnmarshalStrict(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }
    
    return &config, nil
}
```

1.2.3 Implement environment variable substitution (pkg/env/env.go):
```go
var envVarPattern = regexp.MustCompile(`env\(([^)]+)\)`)

func SubstituteEnvVars(input string, envMap map[string]string) (string, error) {
    return envVarPattern.ReplaceAllStringFunc(input, func(match string) string {
        key := strings.TrimSuffix(strings.TrimPrefix(match, "env("), ")")
        if value, exists := envMap[key]; exists {
            return value
        }
        return ""
    }), nil
}

func GetEnvMap() map[string]string {
    env := make(map[string]string)
    for _, e := range os.Environ() {
        if strings.Contains(e, "=") {
            parts := strings.SplitN(e, "=", 2)
            env[parts[0]] = parts[1]
        }
    }
    return env
}
```

### Task 1.3: Pipe-Based Validation System
**Estimated Time**: 1 day

**Subtasks:**
1.3.1 Define pipe interface (pkg/config/pipe.go):
```go
type Pipe interface {
    Default(ctx *Context) error
}

type Context struct {
    Config *Config
    Env    map[string]string
    Logger *logrus.Logger
}

type ValidationPipe struct{}
```

1.3.2 Implement ID uniqueness system (pkg/config/validation/ids.go):
```go
type IDs struct {
    ids  map[string]int
    kind string
}

func New(kind string) *IDs {
    return &IDs{
        ids:  make(map[string]int),
        kind: kind,
    }
}

func (i *IDs) Inc(id string) {
    i.ids[id]++
}

func (i *IDs) Validate() error {
    var errors []string
    for id, count := range i.ids {
        if count > 1 {
            errors = append(errors, fmt.Sprintf("found %d %s with ID '%s', please fix your config", count, i.kind, id))
        }
    }
    if len(errors) > 0 {
        return errors.New(strings.Join(errors, "; "))
    }
    return nil
}
```

1.3.3 Create section validators (pkg/config/validation/*.go):
```go
// build.go
type BuildValidator struct{}

func (BuildValidator) Default(ctx *Context) error {
    if ctx.Config.Build.Configuration == "" {
        return errors.New("build configuration is required")
    }
    if len(ctx.Config.Build.Architectures) == 0 {
        return errors.New("at least one architecture is required")
    }
    return nil
}

// Similar for archive, release, homebrew validators...
```

1.3.4 Implement validation coordinator (pkg/config/pipe.go):
```go
func (ValidationPipe) Run(ctx *Context) error {
    validators := []Pipe{
        &validation.BuildValidator{},
        &validation.ArchiveValidator{},
        &validation.ReleaseValidator{},
        &validation.HomebrewValidator{},
    }
    
    for _, validator := range validators {
        if err := validator.Default(ctx); err != nil {
            return err
        }
    }
    
    return nil
}
```

### Task 1.4: CLI Framework Implementation
**Estimated Time**: 1 day

**Subtasks:**
1.4.1 Create root command (pkg/cli/root.go):
```go
var rootCmd = &cobra.Command{
    Use:   "macreleaser",
    Short: "macOS app release automation",
    Long:  "MacReleaser automates the build, sign, notarize, and release process for macOS applications.",
    Run: func(cmd *cobra.Command, args []string) {
        if err := cmd.Help(); err != nil {
            log.Fatal(err)
        }
    },
}
```

1.4.2 Implement command structure (pkg/cli/*.go):
```go
// init.go
var initCmd = &cobra.Command{
    Use:   "init",
    Short: "Generate example macreleaser configuration",
    Run:   runInit,
}

func runInit(cmd *cobra.Command, args []string) {
    // Generate default .macreleaser.yaml
}

// check.go
var checkCmd = &cobra.Command{
    Use:   "check",
    Short: "Validate configuration file",
    Run:   runCheck,
}

func runCheck(cmd *cobra.Command, args []string) {
    // Load config and run validation
}
```

1.4.3 Create main entry point (cmd/macreleaser/main.go):
```go
func main() {
    if err := rootCmd.Execute(); err != nil {
        log.Fatal(err)
    }
}
```

### Task 1.5: GitHub API Client Setup
**Estimated Time**: 1 day

**Subtasks:**
1.5.1 Basic client setup (pkg/github/client.go):
```go
type Client struct {
    client *github.Client
}

func NewClient(token string) *Client {
    ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
    tc := oauth2.NewClient(context.Background(), ts)
    return &Client{
        client: github.NewClient(tc),
    }
}

func (c *Client) GetRepository(ctx context.Context, owner, repo string) (*github.Repository, error) {
    repo, _, err := c.client.Repositories.Get(ctx, owner, repo)
    return repo, err
}
```

1.5.2 Authentication handling via environment variable:
```go
func GetGitHubToken() string {
    if token := os.Getenv("GITHUB_TOKEN"); token != "" {
        return token
    }
    return ""
}
```

### Task 1.6: Error Handling and Logging
**Estimated Time**: 0.5 days

**Subtasks:**
1.6.1 Set up structured logging (following goreleaser patterns):
```go
func SetupLogger() *logrus.Logger {
    logger := logrus.New()
    logger.SetFormatter(&logrus.TextFormatter{
        FullTimestamp: true,
    })
    return logger
}
```

1.6.2 Implement error message patterns:
```go
func ValidationError(section, field, message string) error {
    return fmt.Errorf("%s.%s: %s (please fix your config)", section, field, message)
}
```

### Task 1.7: Testing Framework
**Estimated Time**: 1 day

**Subtasks:**
1.7.1 Unit tests for configuration parsing:
```go
func TestLoadConfig(t *testing.T) {
    // Test valid config
    // Test missing required fields
    // Test invalid YAML
}
```

1.7.2 Unit tests for validation:
```go
func TestBuildValidator(t *testing.T) {
    tests := []struct {
        name     string
        config   *Config
        expected error
    }{
        // Test cases
    }
    // Run tests
}
```

1.7.3 Unit tests for CLI commands:
```go
func TestInitCommand(t *testing.T) {
    // Test init command generates valid config
}
```

1.7.4 Mock GitHub client for testing:
```go
type MockGitHubClient struct {
    // Mock methods
}
```

1.7.5 Set up test coverage reporting:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Acceptance Criteria

### ✅ Core Functionality
- [ ] `macreleaser init` creates valid `.macreleaser.yaml` with all sections populated
- [ ] `macreleaser check` validates config and reports specific, actionable errors  
- [ ] `macreleaser --version` shows version information  
- [ ] `macreleaser --help` shows all commands and usage  
- [ ] Configuration file loads with proper YAML structure validation
- [ ] Environment variable substitution works with `env(VAR_NAME)` syntax

### ✅ Technical Architecture
- [ ] Pipe-based validation system implemented
- [ ] Cross-section ID uniqueness validation working
- [ ] Environment variable substitution functional
- [ ] GitHub client setup complete with authentication
- [ ] Error messages follow goreleaser patterns
- [ ] Structured logging implemented

### ✅ Build and Distribution
- [ ] Go module builds to single binary with `make build`
- [ ] All tests pass with >80% coverage  
- [ ] Project structure established for future phases
- [ ] Dependencies properly managed in go.mod

### ✅ Documentation
- [ ] README.md with basic usage instructions
- [ ] Example `.macreleaser.yaml` generated by init
- [ ] Code comments for public APIs
- [ ] CLI help text is comprehensive

## Technical Debt and Future Considerations

### Items Deferred to Later Phases
- Full template system for environment variables
- Actual xcodebuild integration (Phase 2)
- Code signing and notarization (Phase 3-4)
- GitHub release creation (Phase 5)
- Homebrew cask generation and submission (Phase 6)

### Architecture Decisions to Revisit
- Consider template system for more complex environment variable scenarios
- Evaluate if additional validation libraries are needed as complexity grows
- Assess logging configuration options (log levels, output formats)

## Dependencies and Tools

### External Go Dependencies
- `github.com/spf13/cobra` - CLI framework
- `github.com/spf13/viper` - CLI configuration
- `github.com/goccy/go-yaml` - YAML parsing
- `github.com/google/go-github` - GitHub API client
- `github.com/sirupsen/logrus` - Structured logging

### Development Tools
- Go 1.20+ for toolchain
- Standard Go tooling (go test, go build, go mod)
- Make for build automation

### External Dependencies (Expected)
- Xcode and Xcode Command Line Tools (not used in Phase 1 but will be required later)
- macOS for development and testing

## Risk Assessment

### Technical Risks
- **Medium**: YAML validation complexity - mitigated by starting with strict parsing
- **Low**: Dependency conflicts - mitigated by using stable, well-maintained libraries
- **Low**: Cross-platform build issues - mitigated by using Go's built-in cross-compilation

### Schedule Risks
- **Low**: Underestimated complexity of pipe architecture - mitigated by following goreleaser patterns
- **Low**: Testing framework complexity - mitigated by starting with basic unit tests

## Success Metrics

- All acceptance criteria completed
- Test coverage >80%
- Code follows Go standard conventions
- Architecture supports future phase requirements
- Documentation is comprehensive for Phase 1 functionality