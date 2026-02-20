package changelog

import (
	"fmt"
	"regexp"

	"github.com/macreleaser/macreleaser/pkg/context"
)

// skipError signals an intentional skip. It satisfies the pipe.IsSkip interface
// checked by the pipeline runner, without importing pkg/pipe (which would cause
// an import cycle through pkg/pipe/registry.go).
type skipError string

func (e skipError) Error() string { return string(e) }
func (e skipError) IsSkip() bool  { return true }

// CheckPipe validates changelog configuration.
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating changelog configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Changelog

	if cfg.Disable {
		return skipError("changelog disabled")
	}

	if cfg.Sort != "" && cfg.Sort != "asc" && cfg.Sort != "desc" {
		return fmt.Errorf("changelog.sort must be \"asc\" or \"desc\", got %q", cfg.Sort)
	}

	for _, pattern := range cfg.Filters.Exclude {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("changelog.filters.exclude: invalid regex %q: %w", pattern, err)
		}
	}

	for _, pattern := range cfg.Filters.Include {
		if _, err := regexp.Compile(pattern); err != nil {
			return fmt.Errorf("changelog.filters.include: invalid regex %q: %w", pattern, err)
		}
	}

	for i, group := range cfg.Groups {
		if group.Title == "" {
			return fmt.Errorf("changelog.groups[%d]: title is required", i)
		}
		if group.Regexp != "" {
			if _, err := regexp.Compile(group.Regexp); err != nil {
				return fmt.Errorf("changelog.groups[%d]: invalid regexp %q: %w", i, group.Regexp, err)
			}
		}
	}

	ctx.Logger.Debug("Changelog configuration validated successfully")
	return nil
}
