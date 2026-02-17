package project

import (
	"fmt"
	"path/filepath"

	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/env"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// CheckPipe validates project configuration
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating project configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Project

	if err := env.CheckResolved(cfg.Name, "project.name"); err != nil {
		return err
	}
	if err := env.CheckResolved(cfg.Scheme, "project.scheme"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Name, "project.name"); err != nil {
		return err
	}
	if !filepath.IsLocal(cfg.Name) {
		return fmt.Errorf("project.name contains a path traversal or absolute path: %q", cfg.Name)
	}

	if err := validate.RequiredString(cfg.Scheme, "project.scheme"); err != nil {
		return err
	}

	ctx.Logger.Debug("Project configuration validated successfully")
	return nil
}
