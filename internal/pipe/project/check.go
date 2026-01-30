package project

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// CheckPipe validates project configuration
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating project configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Project

	if err := validate.RequiredString(cfg.Name, "project.name"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Scheme, "project.scheme"); err != nil {
		return err
	}

	ctx.Logger.Debug("Project configuration validated successfully")
	return nil
}
