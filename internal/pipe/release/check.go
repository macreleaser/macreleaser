package release

import (
	"github.com/macreleaser/macreleaser/pkg/context"
	"github.com/macreleaser/macreleaser/pkg/env"
	"github.com/macreleaser/macreleaser/pkg/validate"
)

// CheckPipe validates release configuration
type CheckPipe struct{}

func (CheckPipe) String() string { return "validating release configuration" }

func (CheckPipe) Run(ctx *context.Context) error {
	if ctx.SkipPublish {
		return skipError("publishing skipped")
	}

	cfg := ctx.Config.Release.GitHub

	if err := env.CheckResolved(cfg.Owner, "release.github.owner"); err != nil {
		return err
	}
	if err := env.CheckResolved(cfg.Repo, "release.github.repo"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Owner, "release.github.owner"); err != nil {
		return err
	}

	if err := validate.RequiredString(cfg.Repo, "release.github.repo"); err != nil {
		return err
	}

	ctx.Logger.Debug("Release configuration validated successfully")
	return nil
}
