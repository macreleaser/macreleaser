package release

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// Pipe validates release configuration
type Pipe struct{}

func (Pipe) String() string { return "validating release configuration" }

func (Pipe) Run(ctx *context.Context) error {
	cfg := ctx.Config.Release.GitHub

	if err := validate.RequiredString(cfg.Owner, "release.github.owner"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Repo, "release.github.repo"); err != nil {
		return err
	}

	ctx.Logger.Debug("Release configuration validated successfully")
	return nil
}
