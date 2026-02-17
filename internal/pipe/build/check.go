package build

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/env"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// CheckPipe validates build configuration
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating build configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Build

	if err := env.CheckResolved(cfg.Configuration, "build.configuration"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Configuration, "build.configuration"); err != nil {
		return err
	}

	ctx.Logger.Debug("Build configuration validated successfully")
	return nil
}
