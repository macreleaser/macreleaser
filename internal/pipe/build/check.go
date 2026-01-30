package build

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// CheckPipe validates build configuration
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating build configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Build

	if err := validate.RequiredString(cfg.Configuration, "build.configuration"); err != nil {
		return err
	}

	if err := validate.RequiredSlice(cfg.Architectures, "build.architectures"); err != nil {
		return err
	}

	validArchs := []string{"arm64", "x86_64", "Universal Binary"}
	if err := validate.AllOneOf(cfg.Architectures, validArchs, "build.architectures"); err != nil {
		return err
	}

	ctx.Logger.Debug("Build configuration validated successfully")
	return nil
}
